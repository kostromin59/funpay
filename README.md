# Funpay

Go library for [Funpay](https://funpay.com/).

> [!important]
> This library is currently developing!

## Example usage

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

  log.Printf("account id: %d", fp.Account().ID())
	log.Printf("username: %q", fp.Account().Username())
	log.Printf("balance: %d", fp.Account().Balance())
	log.Printf("locale: %q", fp.Account().Locale())

  // Load lots for current user
  if err := fp.UpdateLots(context.Background()); err != nil {
    log.Println(err.Error())
    return
  }

  // Returns [nodeID]: []string{offerIDs...}
  lots := fp.Lots().List()
	log.Printf("count of nodes: %d", len(lots))

  // Returns all fields with values to update lot (offer)
  fields, err := fp.LotFields(context.Background(), "", "some_id")
	if err != nil {
		log.Println(err.Error())
		return
	}

  // Change field
  fields["price"] = funpay.LotField{
		Value: "1500",
	}

  // Save lot (offer)
  if err := fp.SaveLot(context.Background(), fields); err != nil {
		log.Println(err.Error())
		return
	}

  // Returns all fields of lot by node (category) without values
  fields, err := fp.LotFields(context.Background(), "some_id", "")
	if err != nil {
		log.Println(err.Error())
		return
	}
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
- [ ] Lots
  - [X] Get fields
  - [X] Get lots
  - [X] Update lot
  - [ ] Remove lots
  - [ ] Create lot
- [ ] Deploy
  - [ ] Deploy into pkg.go.dev
  - [ ] Improve documentation