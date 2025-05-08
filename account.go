package funpay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Account represents an Funpay user session.
// It stores authorization credentials and session cookies.
type Account struct {
	baseURL   string
	goldenKey string
	userAgent string
	csrfToken string
	cookies   []*http.Cookie
	userID    int64
	username  string
	balance   float64
	proxy     *url.URL
	locale    Locale

	mu sync.RWMutex
}

// NewAccount creates a new session instance.
// goldenKey - Funpay authentication token required for API access.
// userAgent - browser User-Agent string to use for requests.
func NewAccount(goldenKey, userAgent string) *Account {
	return &Account{
		goldenKey: goldenKey,
		userAgent: userAgent,
		baseURL:   BaseURL,
	}
}

// GoldenKey returns the account's authentication token.
func (a *Account) GoldenKey() string {
	a.mu.RLock()
	gk := a.goldenKey
	a.mu.RUnlock()
	return gk
}

// UserAgent returns the User-Agent string used for requests.
func (a *Account) UserAgent() string {
	a.mu.RLock()
	ua := a.userAgent
	a.mu.RUnlock()
	return ua
}

// Cookies returns a safe copy of all session cookies.
func (a *Account) Cookies() []*http.Cookie {
	a.mu.RLock()
	c := make([]*http.Cookie, len(a.cookies))
	copy(c, a.cookies)
	a.mu.RUnlock()
	return c
}

// SetCookies updates the account's session cookies.
func (a *Account) SetCookies(cookies []*http.Cookie) {
	a.mu.Lock()
	a.cookies = cookies
	a.mu.Unlock()
}

// SetProxy sets or updates the HTTP proxy for the account's requests.
// To remove proxy and make direct connections, pass nil: account.SetProxy(nil)
func (a *Account) SetProxy(proxy *url.URL) {
	a.mu.Lock()
	a.proxy = proxy
	a.mu.Unlock()
}

// CSRFToken returns the current CSRF token used for request protection.
func (a *Account) CSRFToken() string {
	a.mu.RLock()
	csrfToken := a.csrfToken
	a.mu.RUnlock()
	return csrfToken
}

// Username returns the login name of the Funpay account.
func (a *Account) Username() string {
	a.mu.RLock()
	username := a.username
	a.mu.RUnlock()
	return username
}

// UserID returns the unique identifier of the Funpay account.
// Returns 0 if the account hasn't been updated yet.
func (a *Account) UserID() int64 {
	a.mu.RLock()
	userID := a.userID
	a.mu.RUnlock()
	return userID
}

// Balance returns the current account balance from the badge.
// Returns 0 if the account hasn't been updated yet or balance is zero.
func (a *Account) Balance() float64 {
	a.mu.RLock()
	balance := a.balance
	a.mu.RUnlock()
	return balance
}

// SetBaseURL sets the base URL for Funpay API requests.
// This is primarily used for testing purposes to redirect requests to a mock server.
// Default value is set to [BaseURL] constant when creating a new Account with a [NewAccount] constructor.
func (a *Account) SetBaseURL(baseURL string) {
	a.mu.Lock()
	a.baseURL = baseURL
	a.mu.Unlock()
}

// Locale returns account's locale used in requests.
func (a *Account) Locale() Locale {
	a.mu.RLock()
	locale := a.locale
	a.mu.RUnlock()
	return locale
}

// UpdateLocale makes a new request to update a locale.
func (a *Account) UpdateLocale(ctx context.Context, locale Locale) error {
	const op = "Account.UpdateLocale"

	resp, err := a.Request(ctx, a.baseURL, RequestWithLocale(locale), RequestWithUpdateLocale(true))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.locale = locale

	return nil
}

// Update making a request to get account info.
// Loads userID, username, cookies, csrfToken.
// You should update account info every 40-60 minutes to update cookies.
func (a *Account) Update(ctx context.Context) error {
	const op = "Account.Update"

	resp, err := a.Request(ctx, a.baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	username := strings.TrimSpace(doc.Find(".user-link-name").First().Text())
	rawBalance := doc.Find(".badge-balance").First().Text()
	balanceStr := onlyDigitsRe.ReplaceAllString(rawBalance, "")
	balanceStr = strings.TrimSpace(balanceStr)
	var balance float64
	if balanceStr != "" {
		parsedBalance, err := strconv.ParseFloat(balanceStr, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		balance = parsedBalance
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.username = username
	a.balance = balance
	a.cookies = resp.Cookies()

	return nil
}

// updateAppData extracts data-app-data attribute from body element and sets csrfToken, userID, locale.
func (a *Account) updateAppData(doc *goquery.Document) error {
	const op = "Account.updateAppData"

	appDataRaw, ok := doc.Find("body").Attr("data-app-data")
	if !ok {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	var appData AppData
	if err := json.Unmarshal([]byte(appDataRaw), &appData); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.mu.Lock()
	a.csrfToken = appData.CSRFToken
	a.userID = appData.UserID
	a.locale = appData.Locale
	a.mu.Unlock()

	if appData.UserID == 0 {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	return nil
}

func (a *Account) Request(ctx context.Context, requestURL string, opts ...requestOpt) (*http.Response, error) {
	const op = "Account.Request"

	reqOpts := newRequestOpts()

	if a.proxy != nil {
		opt := RequestWithProxy(a.proxy)
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

	if reqOpts.updateLocale {
		q := reqURL.Query()
		q.Set("setlocale", string(reqOpts.locale))
		reqURL.RawQuery = q.Encode()
	}

	if reqOpts.locale != LocaleRU && !reqOpts.updateLocale {
		path := reqURL.Path
		if path == "" {
			path = "/"
		}

		reqURL.Path = ""
		reqURL = reqURL.JoinPath(string(reqOpts.locale), path)
	}

	req, err := http.NewRequestWithContext(ctx, reqOpts.method, reqURL.String(), reqOpts.body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, c := range reqOpts.cookies {
		req.AddCookie(c)
	}

	req.Header.Set("User-Agent", a.UserAgent())
	for name, value := range reqOpts.headers {
		req.Header.Add(name, value)
	}

	for _, c := range a.Cookies() {
		req.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     GoldenKeyCookie,
		Value:    a.GoldenKey(),
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

	a.SetCookies(resp.Cookies())

	if reqOpts.updateAppData {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		resp.Body = io.NopCloser(bytes.NewReader(b))

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
		if err != nil {
			return resp, fmt.Errorf("%s: %w", op, err)
		}

		if err := a.updateAppData(doc); err != nil {
			return resp, fmt.Errorf("%s: %w", op, err)
		}
	}

	return resp, nil
}
