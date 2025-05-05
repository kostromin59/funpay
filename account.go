package funpay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Account represents an authenticated Funpay user session.
// It stores authorization credentials and session cookies.
type Account struct {
	// goldenKey is the account's authentication token
	// used for authorized requests to Funpay API
	goldenKey string

	// userAgent contains the HTTP User-Agent string
	userAgent string

	// csrfToken stores csrf token from the page
	csrfToken string

	// cookies stores session cookies received from Funpay
	// to maintain authenticated state
	cookies []*http.Cookie

	// TODO: doc
	userID int64

	// TODO: doc
	username string

	mu sync.RWMutex
}

// NewAccount creates a new session instance.
// goldenKey - Funpay authentication token required for API access.
// userAgent - browser User-Agent string to use for requests.
func NewAccount(goldenKey, userAgent string) *Account {
	return &Account{
		goldenKey: goldenKey,
		userAgent: userAgent,
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

func (a *Account) CSRFToken() string {
	a.mu.RLock()
	csrfToken := a.csrfToken
	a.mu.RUnlock()
	return csrfToken
}

func (a *Account) Username() string {
	a.mu.RLock()
	username := a.username
	a.mu.RUnlock()
	return username
}

// Update making a request to get account info.
// You should update account info every 40-60 minutes.
func (a *Account) Update(ctx context.Context) error {
	const op = "Account.Update"

	resp, err := NewRequest(a, FunpayURL).SetContext(ctx).Do()
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

	a.mu.Lock()
	defer a.mu.Unlock()
	username := strings.TrimSpace(doc.Find(".user-link-name").First().Text())

	a.csrfToken = appData.CSRFToken
	a.userID = appData.UserID
	a.username = username
	a.cookies = resp.Cookies()

	return nil
}
