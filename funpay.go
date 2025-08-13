package funpay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	// Domain represents the Funpay website domain.
	Domain = "funpay.com"

	// BaseURL is the base URL for the Funpay website.
	BaseURL = "https://" + Domain
)

var (
	// ErrAccountUnauthorized indicates authentication failure (HTTP 403 Forbidden).
	// Returned when golden key or session cookies are invalid/expired.
	ErrAccountUnauthorized = errors.New("account unauthorized")
)

//go:generate go tool mockgen -destination mocks/funpay.go -package mocks . Funpay
type Funpay interface {
	// UserID returns the unique identifier of the Funpay account.
	// Returns 0 if the account hasn't been updated yet.
	UserID() int64

	// Locale returns the account's locale (see [Locale]). Must be loaded after update.
	Locale() Locale

	// Username returns the account's username. Must be loaded after update.
	Username() string

	// Balance returns the account's balance. Must be loaded after update.
	Balance() int64

	FunpayAuthHandler
	FunpayUpdater
	FunpayRequester
}

type FunpayAuthHandler interface {
	// CSRFToken returns CSRF token extracted from [AppData]. CSRF token updates every call [FunpayRequester.RequestHTML].
	CSRFToken() string

	// UserAgent returns the account's user agent provided into funpay (see [New]).
	GoldenKey() string

	// UserAgent returns the account's user agent provided into funpay (see [New]).
	UserAgent() string
}

type FunpayUpdater interface {
	// BaseURL returns clients baseURL. Needed for tests to substitute the [BaseURL] with test server.
	BaseURL() string

	// SetBaseURL updates clients baseURL. Needed for tests to substitute the [BaseURL] with test server.
	SetBaseURL(baseURL string)

	// Update calls [FunpayRequester.RequestHTML]. You should call it every 40-60 minutes to update PHPSESSIONID cookie.
	// [FunpayRequester.Request] saves all cookies from response if they are not empty.
	Update(ctx context.Context) error

	// UpdateLocale calls [FunpayRequester.RequestHTML] with setlocale query param.
	UpdateLocale(ctx context.Context, locale Locale) error
}

type FunpayRequester interface {
	// Cookies returns a safe copy of all session cookies.
	Cookies() []*http.Cookie

	// SetProxy sets or updates the HTTP proxy for the requests.
	// To remove proxy and make direct connections, pass nil.
	SetProxy(proxy *url.URL)

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
	Request(ctx context.Context, requestURL string, opts ...RequestOpt) (*http.Response, error)

	// RequestHTML calls [FunpayRequester.Request] and converting response as [*goquery.Document].
	// Updates [AppData] and account info (see [Funpay.Account]).
	//
	// Returns nil and [ErrAccountUnauthorized] if [Funpay.UserID] is zero.
	RequestHTML(ctx context.Context, requestURL string, opts ...RequestOpt) (*goquery.Document, error)
}

type FunpayClient struct {
	goldenKey string
	userAgent string
	csrfToken string

	userID   int64
	username string
	balance  int64
	locale   Locale

	baseURL string
	cookies []*http.Cookie
	proxy   *url.URL
	mu      sync.RWMutex
}

// New creates a new instanse of [FunpayClient].
func New(goldenKey, userAgent string) Funpay {
	return &FunpayClient{
		goldenKey: goldenKey,
		userAgent: userAgent,
		baseURL:   BaseURL,
	}
}

// UserID returns the unique identifier of the Funpay account.
// Returns 0 if the account hasn't been updated yet.
func (fp *FunpayClient) UserID() int64 {
	fp.mu.RLock()
	userID := fp.userID
	fp.mu.RUnlock()
	return userID
}

// GoldenKey returns the account's authentication token provided into funpay (see [New]).
func (fp *FunpayClient) GoldenKey() string {
	fp.mu.RLock()
	gk := fp.goldenKey
	fp.mu.RUnlock()
	return gk
}

// UserAgent returns the account's user agent provided into funpay (see [New]).
func (fp *FunpayClient) UserAgent() string {
	fp.mu.RLock()
	ua := fp.userAgent
	fp.mu.RUnlock()
	return ua
}

// Locale returns the account's locale (see [Locale]). Must be loaded after update.
func (fp *FunpayClient) Locale() Locale {
	fp.mu.RLock()
	locale := fp.locale
	fp.mu.RUnlock()
	return locale
}

// Username returns the account's username. Must be loaded after update.
func (fp *FunpayClient) Username() string {
	fp.mu.RLock()
	username := fp.username
	fp.mu.RUnlock()
	return username
}

// Balance returns the account's balance. Must be loaded after update.
func (fp *FunpayClient) Balance() int64 {
	fp.mu.RLock()
	balance := fp.balance
	fp.mu.RUnlock()
	return balance
}

// CSRFToken returns CSRF token extracted from [AppData]. CSRF token updates every call [Funpay.RequestHTML].
func (fp *FunpayClient) CSRFToken() string {
	fp.mu.RLock()
	csrf := fp.csrfToken
	fp.mu.RUnlock()
	return csrf
}

// Cookies returns a safe copy of all session cookies.
func (fp *FunpayClient) Cookies() []*http.Cookie {
	fp.mu.RLock()
	c := make([]*http.Cookie, len(fp.cookies))
	copy(c, fp.cookies)
	fp.mu.RUnlock()
	return c
}

// BaseURL returns baseURL. Needed for tests to substitute the [BaseURL] with test server.
func (fp *FunpayClient) BaseURL() string {
	fp.mu.RLock()
	baseURL := fp.baseURL
	fp.mu.RUnlock()
	return baseURL
}

// SetBaseURL updates baseURL. Needed for tests to substitute the [BaseURL] with test server.
func (fp *FunpayClient) SetBaseURL(baseURL string) {
	fp.mu.Lock()
	fp.baseURL = baseURL
	fp.mu.Unlock()
}

// SetProxy sets or updates the HTTP proxy for the requests.
// To remove proxy and make direct connections, pass nil.
func (fp *FunpayClient) SetProxy(proxy *url.URL) {
	fp.mu.Lock()
	fp.proxy = proxy
	fp.mu.Unlock()
}

// Update calls [Funpay.RequestHTML]. You should call it every 40-60 minutes to update PHPSESSIONID cookie.
// [Funpay.Request] saves all cookies from response if they are not empty.
func (fp *FunpayClient) Update(ctx context.Context) error {
	const op = "Funpay.Update"

	_, err := fp.RequestHTML(ctx, fp.baseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// UpdateLocale calls [Funpay.RequestHTML] with setlocale query param.
func (fp *FunpayClient) UpdateLocale(ctx context.Context, locale Locale) error {
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
func (fp *FunpayClient) Request(ctx context.Context, requestURL string, opts ...RequestOpt) (*http.Response, error) {
	const op = "Funpay.Request"

	reqOpts := NewRequestOpts()

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

	locale := fp.Locale()
	if locale != LocaleRU && reqOpts.method == http.MethodGet {
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

	for _, c := range fp.Cookies() {
		req.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     CookieGoldenKey,
		Value:    fp.GoldenKey(),
		Domain:   "." + Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	req.AddCookie(goldenKeyCookie)

	for _, c := range reqOpts.cookies {
		req.AddCookie(c)
	}

	req.Header.Set(HeaderUserAgent, fp.UserAgent())
	for name, value := range reqOpts.headers {
		req.Header.Add(name, value)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	cookies := resp.Cookies()
	if len(cookies) != 0 {
		fp.mu.Lock()
		fp.cookies = cookies
		fp.mu.Unlock()
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

	return resp, nil
}

// RequestHTML calls [Funpay.Request] and converting response as [*goquery.Document].
// Updates [AppData] and account info (see [Funpay.Account]).
//
// Returns nil and [ErrAccountUnauthorized] if [Funpay.UserID] is zero.
func (fp *FunpayClient) RequestHTML(ctx context.Context, requestURL string, opts ...RequestOpt) (*goquery.Document, error) {
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

	if err := fp.updateUserData(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if fp.UserID() == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	return doc, nil
}

func (fp *FunpayClient) updateUserData(doc *goquery.Document) error {
	const op = "Funpay.updateUserData"
	username := strings.TrimSpace(doc.Find(".user-link-name").First().Text())
	rawBalance := doc.Find(".badge-balance").First().Text()
	balanceStr := onlyDigitsRe.ReplaceAllString(rawBalance, "")
	balanceStr = strings.TrimSpace(balanceStr)
	var balance int64
	if balanceStr != "" {
		parsedBalance, err := strconv.ParseInt(balanceStr, 0, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		balance = parsedBalance
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()
	fp.username = username
	fp.balance = balance

	return nil
}

func (fp *FunpayClient) updateAppData(doc *goquery.Document) error {
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
	fp.userID = appData.UserID
	fp.locale = appData.Locale
	fp.csrfToken = appData.CSRFToken
	fp.mu.Unlock()

	return nil
}
