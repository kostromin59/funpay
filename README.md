# Funpay

Go library for [Funpay](https://funpay.com/).

> [!important]
> This library is currently developing! But may used to handle user and lots.
> Use [Funpay.Request] and [Funpay.RequestHTML] to make your own modules.

## Installation

```sh
go get github.com/kostromin59/funpay
```

## Example usage

### Account handling
```go
func main() {
  goldenKey := "gk"
  ua := "ua"

  fp := funpay.New(goldenKey, ua)
  // Update account info, csrf token and cookies
  if err := fp.Update(context.Background()); err != nil {
		log.Println(err.Error())
		return
	}

 	log.Printf("account id: %d", fp.UserID())
	log.Printf("username: %q", fp.Username())
	log.Printf("balance: %d", fp.Balance())
	log.Printf("locale: %q", fp.Locale())
}
```

### Lots
```go
func main() {
  lots := funpay.NewLots(fp)

  // Load lots for current user
	if err := lots.UpdateLots(context.Background()); err != nil {
		log.Println(err.Error())
		return
	}

	// Returns [nodeID]: []string{offerIDs...}
	lotsList := lots.List()
	log.Printf("count of nodes: %d", len(lotsList))

	// Returns all fields with values to update lot (offer)
	fields, err := lots.LotFields(context.Background(), "", "some_id")
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Change field
	fields["price"] = funpay.LotField{
		Value: "1500",
	}

	// Save lot (offer)
	if err := lots.SaveLot(context.Background(), fields); err != nil {
		log.Println(err.Error())
		return
	}

	// Returns all fields of lot by node (category) without values
  // 2852 - Accounts Call of Duty: Black Ops 6
	fields, err = lots.LotFields(context.Background(), "2852", "")
	if err != nil {
		log.Println(err.Error())
		return
	}

	offerID := fields["offer_id"]
	log.Println(offerID.Value == "0") // true
}
```

## To-Do

> This list may grow while developing.

- [X] Other
  - [X] Use single entrypoint (funpay.New)
- [X] Requests
  - [X] Request with account data
  - [X] Proxy support
  - [X] Locale support (`setlocale` query param and path param for `en` and `uk`)
  - [X] Auto load locale
- [X] Account
  - [X] Info
    - [X] Username
    - [X] Balance (from badge)
  - [X] Updating cookies
  - [X] CSRF Token
  - [X] Substituting base url (for testing)
  - [X] Proxy support
- [ ] Messages
  - [ ] Getting all messages
  - [ ] Getting new messages
  - [ ] Sending
- [X] Lots
  - [X] Get fields
  - [X] Get lots
  - [X] Update lot
  - [X] Delete lot
  - [X] Create lot
- [ ] Deploy
  - [X] Deploy into pkg.go.dev
  - [ ] Improve documentation