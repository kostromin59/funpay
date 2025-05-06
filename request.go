package funpay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	RequestDefaultMethod     = http.MethodGet     // Default HTTP method for requests.
	RequestDefaultTimeout    = 1 * time.Minute    // Default request timeout duration for context.
	RequestUserAgentHeader   = "User-Agent"       // User-Agent header name.
	RequestContentTypeHeader = "Content-Type"     // Content-Type header name.
	RequestJSONContentType   = "application/json" // application/json value for Content-Type header.
	RequestGoldenKeyCookie   = "golden_key"       // Golden key cookie name.
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

// Request represents HTTP request builder for Funpay with account credentials.
type Request struct {
	url     string
	method  string
	body    io.Reader
	cookies []*http.Cookie
	headers map[string]string

	account *Account
	ctx     context.Context
}

// NewRequest creates new API request with default GET method (see [RequestDefaultMethod]).
// You must provide full url. Use [BaseURL] as base of url.
func NewRequest(account *Account, url string) *Request {
	return &Request{
		account: account,
		url:     url,
		method:  RequestDefaultMethod,
	}
}

// SetMethod changes HTTP method for request.
func (r *Request) SetMethod(method string) *Request {
	r.method = method
	return r
}

// SetBody sets request body content.
func (r *Request) SetBody(body io.Reader) *Request {
	r.body = body
	return r
}

// SetCookies adds custom cookies to request. Overrides previous cookies.
func (r *Request) SetCookies(cookies []*http.Cookie) *Request {
	r.cookies = cookies
	return r
}

// SetHeaders adds custom headers to requests. Overrides previous headers.
func (r *Request) SetHeaders(headers map[string]string) *Request {
	r.headers = headers
	return r
}

// SetContext sets context for request cancellation.
// Default context uses [RequestDefaultTimeout] (1 minute) timeout.
func (r *Request) SetContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Do executes configured request with authentication and updates the account cookies.
//
// Returns [*http.Response] and [ErrAccountUnauthorized] if status code equals 403;
// Returns [*http.Response] and [ErrTooManyRequests] if status code equals 429;
// Returns [*http.Response] and [ErrBadStatusCode] if status code equals non-2xx.
//
// Otherwise returns nil and error.
func (r *Request) Do() (*http.Response, error) {
	const op = "Request.Do"

	c := http.DefaultClient

	var ctx context.Context
	if r.ctx != nil {
		ctx = r.ctx
	} else {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), RequestDefaultTimeout)
		ctx = timeoutCtx
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, r.method, r.url, r.body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, c := range r.cookies {
		req.AddCookie(c)
	}

	for name, value := range r.headers {
		req.Header.Add(name, value)
	}

	for _, c := range r.account.Cookies() {
		req.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     RequestGoldenKeyCookie,
		Value:    r.account.GoldenKey(),
		Domain:   "." + Domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	req.AddCookie(goldenKeyCookie)
	req.Header.Set(RequestUserAgentHeader, r.account.UserAgent())

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	r.account.SetCookies(resp.Cookies())

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
