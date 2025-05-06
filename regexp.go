package funpay

import "regexp"

var (
	onlyDigitsRe = regexp.MustCompile(`\D`)
)
