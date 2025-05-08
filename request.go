package funpay

import (
	"io"
	"net/http"
	"net/url"
)

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

func newRequestOpts() *requestOpts {
	return &requestOpts{
		method:        http.MethodGet,
		updateAppData: true,
	}
}

type requestOpt func(options *requestOpts)

func RequestWithMethod(method string) requestOpt {
	return func(options *requestOpts) {
		options.method = method
	}
}

func RequestWithBody(body io.Reader) requestOpt {
	return func(options *requestOpts) {
		options.body = body
	}
}

func RequestWithCookies(cookies []*http.Cookie) requestOpt {
	return func(options *requestOpts) {
		options.cookies = cookies
	}
}

func RequestWithHeaders(headers map[string]string) requestOpt {
	return func(options *requestOpts) {
		options.headers = headers
	}
}

func RequestWithProxy(proxy *url.URL) requestOpt {
	return func(options *requestOpts) {
		options.proxy = proxy
	}
}

func RequestWithLocale(locale Locale) requestOpt {
	return func(options *requestOpts) {
		options.locale = locale
	}
}

func RequestWithUpdateLocale(updateLocale bool) requestOpt {
	return func(options *requestOpts) {
		options.updateLocale = updateLocale
	}
}

func RequestWithUpdateAppData(updateAppData bool) requestOpt {
	return func(options *requestOpts) {
		options.updateAppData = updateAppData
	}
}
