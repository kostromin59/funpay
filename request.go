package funpay

import (
	"io"
	"net/http"
	"net/url"
)

// requestOpts contains configurable parameters for HTTP requests.
// Used internally by Account.Request() to customize request behavior.
type requestOpts struct {
	method        string
	body          io.Reader
	cookies       []*http.Cookie
	headers       map[string]string
	proxy         *url.URL
	locale        Locale
	updateLocale  bool
	updateAppData bool
}

// newRequestOpts creates request options with defaults:
//   - Method: GET
//   - Auto-update app data: enabled
func newRequestOpts() *requestOpts {
	return &requestOpts{
		method:        http.MethodGet,
		updateAppData: true,
	}
}

// requestOpt defines a function type for modifying request options.
type requestOpt func(options *requestOpts)

// RequestWithMethod sets the HTTP method for the request.
// Default: GET
func RequestWithMethod(method string) requestOpt {
	return func(options *requestOpts) {
		options.method = method
	}
}

// RequestWithBody sets the request body.
func RequestWithBody(body io.Reader) requestOpt {
	return func(options *requestOpts) {
		options.body = body
	}
}

// RequestWithCookies adds additional cookies to the request.
// Note: Session cookies are added automatically.
func RequestWithCookies(cookies []*http.Cookie) requestOpt {
	return func(options *requestOpts) {
		options.cookies = cookies
	}
}

// RequestWithHeaders adds custom headers to the request.
// Headers are added in addition to default User-Agent.
func RequestWithHeaders(headers map[string]string) requestOpt {
	return func(options *requestOpts) {
		options.headers = headers
	}
}

// RequestWithProxy overrides the account-level proxy for this request.
// To disable proxy for single request: RequestWithProxy(nil)
func RequestWithProxy(proxy *url.URL) requestOpt {
	return func(options *requestOpts) {
		options.proxy = proxy
	}
}

// RequestWithLocale sets the locale for the request.
// Affects URL path/query based on updateLocale flag.
func RequestWithLocale(locale Locale) requestOpt {
	return func(options *requestOpts) {
		options.locale = locale
	}
}

// RequestWithUpdateLocale controls locale handling:
//   - true: adds locale as query parameter (?setlocale=xx)
//   - false: adds locale as path prefix (/xx/path)
//
// Default behavior depends on account configuration.
func RequestWithUpdateLocale(updateLocale bool) requestOpt {
	return func(options *requestOpts) {
		options.updateLocale = updateLocale
	}
}

// RequestWithUpdateAppData controls automatic app data parsing:
//   - true: extracts CSRF token, user ID etc. from response
//   - false: skips app data processing
//
// Default: true
func RequestWithUpdateAppData(updateAppData bool) requestOpt {
	return func(options *requestOpts) {
		options.updateAppData = updateAppData
	}
}
