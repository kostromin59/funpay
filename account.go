package funpay

import (
	"context"
	"encoding/json"
	"fmt"
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

	_, err := NewRequest(a, a.baseURL).
		SetLocale(locale).
		UpdateLocale(true).
		SetContext(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.locale = locale

	return nil
}

// Update making a request to get account info.
// Loads userID, username, cookies, csrfToken.
// You should update account info every 40-60 minutes.
func (a *Account) Update(ctx context.Context) error {
	const op = "Account.Update"

	resp, err := NewRequest(a, a.baseURL).
		SetContext(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	appDataRaw, ok := doc.Find("body").Attr("data-app-data")
	if !ok {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	var appData AppData
	if err := json.Unmarshal([]byte(appDataRaw), &appData); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if appData.UserID == 0 {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
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

	a.csrfToken = appData.CSRFToken
	a.userID = appData.UserID
	a.locale = appData.Locale
	a.username = username
	a.balance = balance
	a.cookies = resp.Cookies()

	return nil
}
