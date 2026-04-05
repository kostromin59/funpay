package funpay

type opts struct {
	httpClient HTTPClient
	baseURL    string
}

type opt func(opts *opts)

func WithHTTPClient(httpClient HTTPClient) opt {
	return func(opts *opts) {
		opts.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) opt {
	return func(opts *opts) {
		opts.baseURL = baseURL
	}
}
