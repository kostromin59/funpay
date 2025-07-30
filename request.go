package funpay

import (
	"errors"
	"io"
	"net/http"
	"net/url"
)

var (
	// ErrTooManyRequests indicates rate limiting (HTTP 429 Too Many Requests).
	// Returned when exceeding API request limits.
	ErrTooManyRequests = errors.New("too many requests")

	// ErrBadStatusCode indicates unexpected HTTP response status.
	// Returned for any non-2xx status code not covered by other errors.
	ErrBadStatusCode = errors.New("bad status code")
)

var (
	// RequestPostHeaders contains content-type, accept and x-requested-with headers. Copy these values to your headers if needed.
	RequestPostHeaders = map[string]string{
		"content-type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"accept":           "*/*",
		"x-requested-with": "XMLHttpRequest",
	}
)

const (
	// CookieGoldenKey is the cookie name for golden key.
	CookieGoldenKey = "golden_key"

	// HeaderUserAgent is the header name for user agent.
	HeaderUserAgent = "User-Agent"

	FormCSRFToken = "csrf_token"
)

// RequestOpts contains configurable parameters for HTTP requests.
// Used internally by [Funpay.Request] to customize request behavior.
type RequestOpts struct {
	method  string
	body    io.Reader
	cookies []*http.Cookie
	headers map[string]string
	proxy   *url.URL
}

// NewRequestOpts creates request options with defaults:
//   - Method: GET
func NewRequestOpts() *RequestOpts {
	return &RequestOpts{
		method: http.MethodGet,
	}
}

// RequestOpt defines a function type for modifying request options.
type RequestOpt func(options *RequestOpts)

// RequestWithMethod sets the HTTP method for the request.
// Default: GET
func RequestWithMethod(method string) RequestOpt {
	return func(options *RequestOpts) {
		options.method = method
	}
}

// RequestWithBody sets the request body.
func RequestWithBody(body io.Reader) RequestOpt {
	return func(options *RequestOpts) {
		options.body = body
	}
}

// RequestWithCookies adds additional cookies to the request.
// Note: Session cookies are added automatically.
func RequestWithCookies(cookies []*http.Cookie) RequestOpt {
	return func(options *RequestOpts) {
		options.cookies = cookies
	}
}

// RequestWithHeaders adds custom headers to the request.
// Headers are added in addition to default User-Agent.
func RequestWithHeaders(headers map[string]string) RequestOpt {
	return func(options *RequestOpts) {
		options.headers = headers
	}
}

// RequestWithProxy overrides the account-level proxy for this request.
// To disable proxy for single request: RequestWithProxy(nil)
func RequestWithProxy(proxy *url.URL) RequestOpt {
	return func(options *RequestOpts) {
		options.proxy = proxy
	}
}
