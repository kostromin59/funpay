package funpay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	// Domain represents the Funpay website domain.
	Domain = "funpay.com"

	// BaseURL is the base URL for the Funpay website.
	BaseURL = "https://" + Domain
)

type Funpay struct {
	account *account

	csrfToken string
	baseURL   string
	cookies   []*http.Cookie
	proxy     *url.URL
	mu        sync.RWMutex
}

// New creates a new instanse of [Funpay].
func New(goldenKey, userAgent string) *Funpay {
	return &Funpay{
		account: newAccount(goldenKey, userAgent),
		baseURL: BaseURL,
	}
}

// Update calls [Funpay.RequestHTML].
func (fp *Funpay) Update(ctx context.Context) error {
	const op = "Funpay.Update"

	_, err := fp.RequestHTML(ctx, fp.baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UpdateLocale calls [Funpay.RequestHTML] with setlocale query param.
func (fp *Funpay) UpdateLocale(ctx context.Context, locale Locale) error {
	const op = "Funpay.UpdateLocale"

	reqURL, err := url.Parse(fp.baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q := reqURL.Query()
	q.Set("setlocale", string(locale))
	reqURL.RawQuery = q.Encode()

	if _, err := fp.RequestHTML(ctx, reqURL.String()); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Account returns account info: id, username, balance, locale. [Funpay.RequestHTML] updates account info.
func (fp *Funpay) Account() *account {
	return fp.account
}

// Cookies returns a safe copy of all session cookies.
func (fp *Funpay) Cookies() []*http.Cookie {
	fp.mu.RLock()
	c := make([]*http.Cookie, len(fp.cookies))
	copy(c, fp.cookies)
	fp.mu.RUnlock()
	return c
}

// SetBaseURL updates baseURL. It is not concurrency safe. Needed for tests.
func (fp *Funpay) SetBaseURL(baseURL string) {
	fp.baseURL = baseURL
}

// SetProxy sets or updates the HTTP proxy for the requests.
// To remove proxy and make direct connections, pass nil.
func (fp *Funpay) SetProxy(proxy *url.URL) {
	fp.mu.Lock()
	fp.proxy = proxy
	fp.mu.Unlock()
}

// Request executes an HTTP request using the account's session.
//
// It handles:
//   - Proxy configuration (if set),
//   - Locale settings (path or query param),
//   - Cookie management (session and golden key),
//   - User-Agent header,
//   - Response status code validation,
//
// Specific returns:
//   - [*http.Response] and [ErrAccountUnauthorized] if status code equals 403,
//   - [*http.Response] and [ErrToManyRequests] if status code equals 429,
//   - [*http.Response] [ErrBadStatusCode] otherwise.
func (fp *Funpay) Request(ctx context.Context, requestURL string, opts ...requestOpt) (*http.Response, error) {
	const op = "Funpay.Request"

	reqOpts := newRequestOpts()

	if fp.proxy != nil {
		opt := RequestWithProxy(fp.proxy)
		opt(reqOpts)
	}

	for _, opt := range opts {
		opt(reqOpts)
	}

	t := &http.Transport{}
	if reqOpts.proxy != nil {
		t.Proxy = http.ProxyURL(reqOpts.proxy)
	}

	c := http.DefaultClient
	c.Transport = t

	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	locale := fp.Account().Locale()
	if locale != LocaleRU {
		path := reqURL.Path
		if path == "" {
			path = "/"
		}

		reqURL.Path = ""
		reqURL = reqURL.JoinPath(string(locale), path)
	}

	req, err := http.NewRequestWithContext(ctx, reqOpts.method, reqURL.String(), reqOpts.body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, c := range reqOpts.cookies {
		req.AddCookie(c)
	}

	req.Header.Set(HeaderUserAgent, fp.Account().UserAgent())
	for name, value := range reqOpts.headers {
		req.Header.Add(name, value)
	}

	for _, c := range fp.Cookies() {
		req.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     CookieGoldenKey,
		Value:    fp.Account().GoldenKey(),
		Domain:   "." + Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	req.AddCookie(goldenKeyCookie)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if resp.StatusCode == 403 {
			return resp, fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
		}

		if resp.StatusCode == 429 {
			return resp, fmt.Errorf("%s: %w", op, ErrTooManyRequests)
		}

		return resp, fmt.Errorf("%s: %w (%d)", op, ErrBadStatusCode, resp.StatusCode)
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.cookies = resp.Cookies()

	return resp, nil
}

// RequestHTML calls [Funpay.Request] and converting response as [*goquery.Document].
// Updates [AppData] and account info (see [Funpay.Account]).
//
// Returns nil and [ErrAccountUnauthorized] if [account.ID] is zero.
func (fp *Funpay) RequestHTML(ctx context.Context, requestURL string, opts ...requestOpt) (*goquery.Document, error) {
	const op = "Funpay.RequestHTML"

	resp, err := fp.Request(ctx, requestURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := fp.updateAppData(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := fp.Account().update(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if fp.Account().ID() == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	return doc, nil
}

func (fp *Funpay) updateAppData(doc *goquery.Document) error {
	const op = "Funpay.updateAppData"

	appDataRaw, ok := doc.Find("body").Attr("data-app-data")
	if !ok {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	var appData AppData
	if err := json.Unmarshal([]byte(appDataRaw), &appData); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	fp.account.setID(appData.UserID)
	fp.Account().setLocale(appData.Locale)

	fp.mu.Lock()
	fp.csrfToken = appData.CSRFToken
	fp.mu.Unlock()

	return nil
}
