package funpay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	RequestDefaultMethod  = http.MethodGet
	RequestDefaultTimeout = 1 * time.Minute

	RequestUserAgentHeader = "User-Agent"

	RequestGoldenKeyCookie = "golden_key"
)

type Request struct {
	url     string
	method  string
	body    io.Reader
	cookies []*http.Cookie
	headers map[string]string

	account *Account
	ctx     context.Context
}

func NewRequest(account *Account, url string) *Request {
	return &Request{
		account: account,
		url:     url,
		method:  RequestDefaultMethod,
	}
}

func (r *Request) SetMethod(method string) *Request {
	r.method = method
	return r
}

func (r *Request) SetBody(body io.Reader) *Request {
	r.body = body
	return r
}

func (r *Request) SetCookies(cookies []*http.Cookie) *Request {
	r.cookies = cookies
	return r
}

func (r *Request) SetHeaders(headers map[string]string) *Request {
	r.headers = headers
	return r
}

func (r *Request) SetContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

func (r *Request) Do() (*http.Response, error) {
	const op = "request.Do"

	c := http.DefaultClient

	ctx := context.Background()
	if r.ctx != nil {
		ctx = r.ctx
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
		Domain:   "." + FunpayDomain,
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

	return resp, nil
}
