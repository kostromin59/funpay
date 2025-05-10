package funpay

// AppData represents the object from data-app-data attribute inside the body element.
type AppData struct {
	CSRFToken string `json:"csrf-token"`
	UserID    int64  `json:"userId"`
	Locale    Locale `json:"locale"`
}
