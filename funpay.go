package funpay

import "errors"

const (
	// Domain represents the Funpay website domain.
	Domain = "funpay.com"

	// BaseURL is the base URL for the Funpay website.
	BaseURL = "https://" + Domain

	// GoldenKeyCookie is the cookie name for golden key.
	GoldenKeyCookie = "golden_key"
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

// AppData represents the object from data-app-data attribute inside the body element.
type AppData struct {
	CSRFToken string `json:"csrf-token,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	Locale    Locale `json:"locale,omitempty"`
}

// Locale represents the Funpay webiste locale.
type Locale string

const (
	LocaleRU Locale = "ru"
	LocaleEN Locale = "en"
	LocaleUK Locale = "uk"
)
