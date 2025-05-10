# Funpay

Go library for [Funpay](https://funpay.com/).

> [!important]
> This library is currently developing!

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
- [ ] Selling goods
  - [ ] Get selling list
  - [ ] Update goods
  - [ ] Remove goods
  - [ ] Creating goods
- [ ] Deploy
  - [ ] Deploy into pkg.go.dev
  - [ ] Improve documentation