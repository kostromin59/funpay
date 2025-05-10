package funpay

import (
	"context"
	"encoding/json"
	"errors"
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

	// GoldenKeyCookie is the cookie name for golden key.
	GoldenKeyCookie = "golden_key"
)

var (
	// ErrAccountUnauthorized indicates authentication failure (HTTP 403 Forbidden).
	// Returned when golden key or session cookies are invalid/expired.
	ErrAccountUnauthorized = errors.New("account unauthorized")

	// ErrTooManyRequests indicates rate limiting (HTTP 429 Too Many Requests).
	// Returned when exceeding API request limits.
	ErrTooManyRequests = errors.New("too many requests")

	// ErrBadStatusCode indicates unexpected HTTP response status.
	// Returned for any non-2xx status code not covered by other errors.
	ErrBadStatusCode = errors.New("bad status code")
)

// AppData represents the object from data-app-data attribute inside the body element.
type AppData struct {
	CSRFToken string `json:"csrf-token,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	Locale    Locale `json:"locale,omitempty"`
}

type Funpay struct {
	account *account

	locale    Locale
	csrfToken string

	baseURL string
	cookies []*http.Cookie
	proxy   *url.URL
	mu      sync.RWMutex
}

func New(goldenKey, userAgent string) *Funpay {
	return &Funpay{
		account: newAccount(goldenKey, userAgent),
		baseURL: BaseURL,
	}
}

func (fp *Funpay) Locale() Locale {
	return fp.locale
}

func (fp *Funpay) Update(ctx context.Context) error {
	const op = "Funpay.Update"
	doc, err := fp.RequestHTML(ctx, fp.baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := fp.account.update(doc); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

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

	fp.mu.Lock()
	fp.csrfToken = appData.CSRFToken
	fp.locale = appData.Locale
	fp.account.setID(appData.UserID)
	fp.mu.Unlock()

	if appData.UserID == 0 {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	return nil
}

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

	// if reqOpts.updateLocale {
	// 	q := reqURL.Query()
	// 	q.Set("setlocale", string(reqOpts.locale))
	// 	reqURL.RawQuery = q.Encode()
	// }

	if fp.locale != LocaleRU {
		path := reqURL.Path
		if path == "" {
			path = "/"
		}

		reqURL.Path = ""
		reqURL = reqURL.JoinPath(string(fp.locale), path)
	}

	req, err := http.NewRequestWithContext(ctx, reqOpts.method, reqURL.String(), reqOpts.body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, c := range reqOpts.cookies {
		req.AddCookie(c)
	}

	req.Header.Set("User-Agent", fp.Account().UserAgent())
	for name, value := range reqOpts.headers {
		req.Header.Add(name, value)
	}

	for _, c := range fp.Cookies() {
		req.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     GoldenKeyCookie,
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

	return doc, nil
}
