package funpay

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type account struct {
	goldenKey string
	userAgent string

	id       int64
	username string
	balance  float64

	mu sync.RWMutex
}

func newAccount(goldenKey, userAgent string) *account {
	return &account{
		goldenKey: goldenKey,
		userAgent: userAgent,
	}
}

// GoldenKey returns the account's authentication token.
func (a *account) GoldenKey() string {
	a.mu.RLock()
	gk := a.goldenKey
	a.mu.RUnlock()
	return gk
}

func (a *account) Username() string {
	a.mu.RLock()
	username := a.username
	a.mu.RUnlock()
	return username
}

func (a *account) Balance() float64 {
	a.mu.RLock()
	balance := a.balance
	a.mu.RUnlock()
	return balance
}

func (a *account) UserAgent() string {
	a.mu.RLock()
	ua := a.userAgent
	a.mu.RUnlock()
	return ua
}

func (a *account) setID(id int64) {
	a.mu.Lock()
	a.id = id
	a.mu.Unlock()
}

// ID returns the unique identifier of the Funpay account.
// Returns 0 if the account hasn't been updated yet.
func (a *account) ID() int64 {
	a.mu.RLock()
	userID := a.id
	a.mu.RUnlock()
	return userID
}

func (a *account) update(doc *goquery.Document) error {
	const op = "account.Update"

	username := strings.TrimSpace(doc.Find(".user-link-name").First().Text())
	rawBalance := doc.Find(".badge-balance").First().Text()
	balanceStr := onlyDigitsRe.ReplaceAllString(rawBalance, "")
	balanceStr = strings.TrimSpace(balanceStr)
	var balance float64
	if balanceStr != "" {
		parsedBalance, err := strconv.ParseFloat(balanceStr, 64)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		balance = parsedBalance
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.username = username
	a.balance = balance

	return nil
}
