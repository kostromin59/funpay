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

type (
	// NodeID represents ID of category.
	NodeID string
	// OfferID represents ID of offer.
	OfferID string
)

type (
	// FieldKey represents key type for [Fields].
	FieldKey string
	// Fields represents type contains fields from edit lot page.
	Fields map[FieldKey]Field
)

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

	// Fields loads [Fields] for [OfferID]. Values will be filled with provided offerID.
	FieldsByOfferID(ctx context.Context, offerID OfferID) (Fields, error)

	// FieldsByNodeID loads [Fields] for [NodeID].
	FieldsByNodeID(ctx context.Context, nodeID NodeID) (Fields, error)

	// ByUser gets lots for provided userID. Key represents nodeID, value represents slice of offerIDs.
	ByUser(ctx context.Context, userID int64) (map[NodeID][]OfferID, error)

	// Update updates lots for current account. Use [Lots.List] to get loaded lots.
	// Returns [funpay.ErrAccountUnauthorized] if user id equals 0. Call [Funpay.Update] to update account info.
	Update(ctx context.Context) error

	// List returns loaded lots (see [Lots.Update]).
	List() map[NodeID][]OfferID
}

type LotsClient struct {
	fp funpay.Funpay

	list map[NodeID][]OfferID
	mu   sync.RWMutex
}

func New(fp funpay.Funpay) Lots {
	return &LotsClient{
		fp: fp,
	}
}

func (l *LotsClient) Save(ctx context.Context, fields Fields) error {
	const op = "LotsClient.Save"

	body := url.Values{}

	for name, v := range fields {
		body.Set(string(name), v.Value)
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

func (l *LotsClient) FieldsByOfferID(ctx context.Context, offerID OfferID) (Fields, error) {
	const op = "LotsClient.FieldsByOfferID"

	fields, err := l.fields(ctx, "", offerID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return fields, nil
}

func (l *LotsClient) FieldsByNodeID(ctx context.Context, nodeID NodeID) (Fields, error) {
	const op = "LotsClient.FieldsByOfferID"

	fields, err := l.fields(ctx, nodeID, "")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return fields, nil
}

func (l *LotsClient) fields(ctx context.Context, nodeID NodeID, offerID OfferID) (Fields, error) {
	const op = "LotsClient.fields"

	reqURL, err := url.Parse(l.fp.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	reqURL = reqURL.JoinPath("lots", "offerEdit")

	q := reqURL.Query()
	if offerID != "" {
		q.Set("offer", string(offerID))
	}
	if nodeID != "" {
		q.Set("node", string(nodeID))
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

			fields[FieldKey(name)] = field

		default:
			if name == funpay.FormCSRFToken {
				return
			}

			value := s.AttrOr("value", "")
			fields[FieldKey(name)] = Field{
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
		fields[FieldKey(name)] = Field{
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

		fields[FieldKey(name)] = field
	})

	return fields
}

func (l *LotsClient) ByUser(ctx context.Context, userID int64) (map[NodeID][]OfferID, error) {
	const op = "LotsClient.ByUser"

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

func (l *LotsClient) extractLots(doc *goquery.Document) (map[NodeID][]OfferID, error) {
	const op = "LotsClient.extractLots"

	lots := make(map[NodeID][]OfferID)

	offerUrls := doc.Find(".offer")
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
		offerIDs := make([]OfferID, 0, urlElements.Length())
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

			offerIDs = append(offerIDs, OfferID(offerID))
		})

		lots[NodeID(pathComponents[2])] = offerIDs
	}

	return lots, nil
}

func (l *LotsClient) Update(ctx context.Context) error {
	const op = "LotsClient.Update"

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

func (l *LotsClient) List() map[NodeID][]OfferID {
	l.mu.RLock()
	list := l.list
	l.mu.RUnlock()
	return list
}

func (l *LotsClient) updateList(list map[NodeID][]OfferID) {
	l.mu.Lock()
	l.list = list
	l.mu.Unlock()
}
