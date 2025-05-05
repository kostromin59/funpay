package funpay

const (
	// FunpayDomain represents the Funpay website domain.
	FunpayDomain = "funpay.com"

	// FunpayURL is the base URL for the Funpay website.
	FunpayURL = "https://" + FunpayDomain
)

// AppData represents the object from data-app-data attribute inside the body element.
type AppData struct {
	CSRFToken string `json:"csrf-token,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	Locale    string `json:"locale,omitempty"`
}
