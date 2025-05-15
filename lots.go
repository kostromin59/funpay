package funpay

type LotFields map[string]LotField

type LotField struct {
	Value    string   `json:"value"`
	Variants []string `json:"variants"`
}
