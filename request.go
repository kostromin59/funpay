package funpay

import (
	"net/http"
	"sync"
)

const (
	// DefaultFunpayURL is the base URL for the Funpay website.
	DefaultFunpayURL = "https://funpay.com"
)

// Request represents an HTTP request to funpay with protected fields that can be accessed concurrently.
// All exported methods are thread-safe.
type Request struct {
	// goldenKey is an authentication key
	goldenKey string

	// userAgent identifies the browser logged in
	userAgent string

	// cookies stores funpay HTTP response cookies
	cookies []*http.Cookie

	mu sync.RWMutex
}

// NewRequest creates a new Request instance with the given goldenKey and userAgent.
func NewRequest(goldenKey, userAgent string) *Request {
	return &Request{
		goldenKey: goldenKey,
		userAgent: userAgent,
	}
}

// GoldenKey returns the authentication key.
func (r *Request) GoldenKey() string {
	// Defer breaks inline optimisation
	r.mu.RLock()
	gk := r.goldenKey
	r.mu.Unlock()

	return gk
}

// UserAgent returns the HTTP User-Agent header value that will be set when creating the Request.
func (r *Request) UserAgent() string {
	// Defer breaks inline optimisation
	r.mu.RLock()
	ua := r.userAgent
	r.mu.Unlock()

	return ua
}

// Cookies returns a copy of all cookies.
func (r *Request) Cookies() []*http.Cookie {
	// Defer breaks inline optimisation
	r.mu.RLock()
	c := make([]*http.Cookie, len(r.cookies)) // Copy resolves race condition
	_ = copy(c, r.cookies)
	r.mu.Unlock()

	return c
}
