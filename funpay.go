package funpay

const (
	// Domain represents the Funpay website domain.
	Domain = "funpay.com"

	// BaseURL is the base URL for the Funpay website.
	BaseURL = "https://" + Domain
)

// AppData represents the object from data-app-data attribute inside the body element.
type AppData struct {
	CSRFToken string `json:"csrf-token,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	Locale    string `json:"locale,omitempty"`
}
