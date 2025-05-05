package funpay

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	RequestUserAgentHeader = "User-Agent"

	RequestGoldenKeyCookie = "golden_key"
)

type Request struct {
	*http.Request
	account *Account
}

func NewRequest(account *Account, method, path string, body io.Reader) (*Request, error) {
	const op = "request.NewRequest"

	reqURL, err := url.Parse(FunpayURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reqURL = reqURL.JoinPath(path)

	rawReq, err := http.NewRequest(method, reqURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	req := &Request{
		rawReq,
		account,
	}

	return req, nil
}

func (r *Request) Do() (*http.Response, error) {
	const op = "request.Do"

	for _, c := range r.account.Cookies() {
		r.AddCookie(c)
	}

	goldenKeyCookie := &http.Cookie{
		Name:     RequestGoldenKeyCookie,
		Value:    r.account.GoldenKey(),
		Domain:   "." + FunpayDomain,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	r.AddCookie(goldenKeyCookie)
	r.Header.Set(RequestUserAgentHeader, r.account.UserAgent())

	c := http.DefaultClient

	resp, err := c.Do(r.Request)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	r.account.SetCookies(resp.Cookies())

	return resp, nil
}
