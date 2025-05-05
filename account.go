package funpay

import (
	"net/http"
	"sync"
)

// Account represents an authenticated Funpay user session.
// It stores authorization credentials and session cookies.
type Account struct {
	// goldenKey is the account's authentication token
	// used for authorized requests to Funpay API
	goldenKey string

	// userAgent contains the HTTP User-Agent string
	userAgent string

	// cookies stores session cookies received from Funpay
	// to maintain authenticated state
	cookies []*http.Cookie

	mu sync.RWMutex
}

// NewAccount creates a new session instance.
// goldenKey - Funpay authentication token required for API access
// userAgent - browser User-Agent string to use for requests
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

func (a *Account) SetCookies(cookies []*http.Cookie) {
	a.mu.Lock()
	a.cookies = cookies
	a.mu.Unlock()
}
