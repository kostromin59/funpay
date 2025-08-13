package lots

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/kostromin59/funpay"
)

// Fields represents type contains fields from edit lot page.
type Fields map[string]Field

// Field represents type contains description of the field on edit lot page.
type Field struct {
	Value    string   `json:"value"`
	Variants []string `json:"variants"`
}

//go:generate go tool mockgen -destination ../mocks/lots.go -package mocks . Lots
type Lots interface {
	// Save makes request to /lots/offerSave. Use [Lots.Fields] to get fields.
	//
	//	Fields:
	//	- Provide offer_id to update lot;
	//	- Set offer_id = "0" to create lot;
	//	- Set deleted = "1" to delete lot.
	Save(ctx context.Context, fields Fields) error

	// Fields loads [Fields] for nodeID (category) or offerID. Values will be filled with provided offerID.
	Fields(ctx context.Context, nodeID, offerID string) (Fields, error)

	// ByUser gets lots for provided userID. Key represents nodeID, value represents slice of offerIDs.
	ByUser(ctx context.Context, userID int64) (map[string][]string, error)

	// Update updates lots for current account. Use [Lots.List] to get loaded lots.
	// Returns [funpay.ErrAccountUnauthorized] if user id equals 0. Call [Funpay.Update] to update account info.
	Update(ctx context.Context) error

	// List returns loaded lots (see [Lots.Update]) in format nodeID: slice of offerIDs.
	List() map[string][]string
}

type LotsClient struct {
	fp funpay.Funpay

	list map[string][]string
	mu   sync.RWMutex
}

func New(fp funpay.Funpay) Lots {
	return &LotsClient{
		fp: fp,
	}
}

// Save makes request to /lots/offerSave. Use [Lots.Fields] to get fields.
//
//	Fields:
//	- Provide offer_id to update lot;
//	- Set offer_id = "0" to create lot;
//	- Set deleted = "1" to delete lot.
func (l *LotsClient) Save(ctx context.Context, fields Fields) error {
	const op = "Lots.Save"

	body := url.Values{}

	for name, v := range fields {
		body.Set(name, v.Value)
	}

	body.Set(funpay.FormCSRFToken, l.fp.CSRFToken())
	body.Set("location", "trade")

	_, err := l.fp.Request(ctx, l.fp.BaseURL()+"/lots/offerSave",
		funpay.RequestWithMethod(http.MethodPost),
		funpay.RequestWithBody(bytes.NewBufferString(body.Encode())),
		funpay.RequestWithHeaders(funpay.RequestPostHeaders),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Fields loads [Fields] for nodeID (category) or offerID. Values will be filled with provided offerID.
func (l *LotsClient) Fields(ctx context.Context, nodeID, offerID string) (Fields, error) {
	const op = "Lots.Fields"

	reqURL, err := url.Parse(l.fp.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reqURL = reqURL.JoinPath("lots", "offerEdit")

	q := reqURL.Query()
	if offerID != "" {
		q.Set("offer", offerID)
	}
	if nodeID != "" {
		q.Set("node", nodeID)
	}
	reqURL.RawQuery = q.Encode()

	doc, err := l.fp.RequestHTML(ctx, reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return l.extractFields(doc), nil
}

func (l *LotsClient) extractFields(doc *goquery.Document) Fields {
	fields := make(Fields)
	form := doc.Find("form")
	form.Find("input[name]").Each(func(i int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok {
			return
		}

		switch s.AttrOr("type", "") {
		case "checkbox":
			field := Field{
				Variants: []string{"on"},
			}
			_, ok := s.Attr("checked")
			if ok {
				field.Value = "on"
			}

			fields[name] = field

		default:
			if name == funpay.FormCSRFToken {
				return
			}

			value := s.AttrOr("value", "")
			fields[name] = Field{
				Value: value,
			}
		}
	})

	form.Find("textarea[name]").Each(func(i int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok {
			return
		}

		value := s.Text()
		fields[name] = Field{
			Value: value,
		}
	})

	form.Find("select[name]").Each(func(i int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok {
			return
		}

		field := Field{}

		opts := s.Find("option[value]")
		variants := make([]string, 0, opts.Length())
		opts.Each(func(i int, s *goquery.Selection) {
			value, ok := s.Attr("value")
			if !ok {
				return
			}

			if value == "" {
				return
			}

			variants = append(variants, value)

			if _, ok := s.Attr("selected"); ok {
				field.Value = value
			}
		})

		field.Variants = variants

		fields[name] = field
	})

	return fields
}

// ByUser gets lots for provided userID. Key represents nodeID, value represents slice of offerIDs.
func (l *LotsClient) ByUser(ctx context.Context, userID int64) (map[string][]string, error) {
	const op = "Lots.ByUser"

	reqURL, err := url.Parse(l.fp.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reqURL = reqURL.JoinPath("users", fmt.Sprintf("%d", userID), "/")

	doc, err := l.fp.RequestHTML(ctx, reqURL.String())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lots, err := l.extractLots(doc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return lots, nil
}

// Key represents nodeID, value represents slice of offerIDs.
func (l *LotsClient) extractLots(doc *goquery.Document) (map[string][]string, error) {
	const op = "Lots.extractLots"

	offerUrls := doc.Find(".offer")
	lots := make(map[string][]string)
	for _, offer := range offerUrls.EachIter() {
		nodeHref, ok := offer.Find("h3 a[href]").Attr("href")
		if !ok {
			continue
		}

		nodeURL, err := url.Parse(nodeHref)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		pathComponents := strings.Split(nodeURL.Path, "/")
		if len(pathComponents) < 3 {
			continue
		}

		urlElements := offer.Find("a.tc-item[href]")
		offerIDs := make([]string, 0, urlElements.Length())
		urlElements.Each(func(i int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if !ok {
				return
			}

			rawURL, err := url.Parse(href)
			if err != nil {
				return
			}

			q := rawURL.Query()
			offerID := q.Get("id")

			offerIDs = append(offerIDs, offerID)
		})

		lots[pathComponents[2]] = offerIDs
	}

	return lots, nil
}

// Update updates lots for current account. Use [Lots.List] to get loaded lots.
// Returns [funpay.ErrAccountUnauthorized] if user id equals 0. Call [Funpay.Update] to update account info.
func (l *LotsClient) Update(ctx context.Context) error {
	const op = "Lots.Update"

	id := l.fp.UserID()
	if id == 0 {
		return fmt.Errorf("%s: %w", op, funpay.ErrAccountUnauthorized)
	}

	lots, err := l.ByUser(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	l.updateList(lots)

	return nil
}

// List returns loaded lots (see [Lots.Update]) in format nodeID: slice of offerIDs.
func (l *LotsClient) List() map[string][]string {
	l.mu.RLock()
	list := l.list
	l.mu.RUnlock()
	return list
}

func (l *LotsClient) updateList(list map[string][]string) {
	l.mu.Lock()
	l.list = list
	l.mu.Unlock()
}
