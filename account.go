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
func (r *Account) GoldenKey() string {
	r.mu.RLock()
	gk := r.goldenKey
	r.mu.RUnlock()
	return gk
}

// UserAgent returns the User-Agent string used for requests.
func (r *Account) UserAgent() string {
	r.mu.RLock()
	ua := r.userAgent
	r.mu.RUnlock()
	return ua
}

// Cookies returns a safe copy of all session cookies.
func (r *Account) Cookies() []*http.Cookie {
	r.mu.RLock()
	c := make([]*http.Cookie, len(r.cookies))
	copy(c, r.cookies)
	r.mu.RUnlock()
	return c
}
