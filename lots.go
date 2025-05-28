package funpay

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// LotFields represents type contains fields from edit lot page.
type LotFields map[string]LotField

// LotField represents type contains description of the field on edit lot page.
type LotField struct {
	Value    string   `json:"value"`
	Variants []string `json:"variants"`
}

type Lots struct {
	fp *Funpay

	list map[string][]string
	mu   sync.RWMutex
}

func NewLots(fp *Funpay) *Lots {
	return &Lots{
		fp: fp,
	}
}

// SaveLot makes request to /lots/offerSave. Use [Lots.LotFields] to get fields.
//
//	Fields:
//	- Provide offer_id to update lot;
//	- Set offer_id = "0" to create lot;
//	- Set deleted = "1" to delete lot.
func (l *Lots) SaveLot(ctx context.Context, fields LotFields) error {
	const op = "Lots.SaveLot"

	body := url.Values{}

	for name, v := range fields {
		body.Set(name, v.Value)
	}

	body.Set(FormCSRFToken, l.fp.CSRFToken())
	body.Set("location", "trade")

	_, err := l.fp.Request(ctx, l.fp.BaseURL()+"/lots/offerSave",
		RequestWithMethod(http.MethodPost),
		RequestWithBody(bytes.NewBufferString(body.Encode())),
		RequestWithHeaders(RequestPostHeaders),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// LotFields loads [LotFields] for nodeID (category) or offerID. Values will be filled with provided offerID.
func (l *Lots) LotFields(ctx context.Context, nodeID, offerID string) (LotFields, error) {
	const op = "Lots.LotFields"

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

func (l *Lots) extractFields(doc *goquery.Document) LotFields {
	fields := make(LotFields)
	form := doc.Find("form")
	form.Find("input[name]").Each(func(i int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok {
			return
		}

		switch s.AttrOr("type", "") {
		case "checkbox":
			field := LotField{
				Variants: []string{"on", ""},
			}
			_, ok := s.Attr("checked")
			if ok {
				field.Value = "on"
			}

			fields[name] = field

		default:
			if name == FormCSRFToken {
				return
			}

			value := s.AttrOr("value", "")
			fields[name] = LotField{
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
		fields[name] = LotField{
			Value: value,
		}
	})

	form.Find("select[name]").Each(func(i int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok {
			return
		}

		field := LotField{}

		opts := s.Find("option[value]")
		variants := make([]string, 0, opts.Length())
		opts.Each(func(i int, s *goquery.Selection) {
			value, ok := s.Attr("value")
			if !ok {
				return
			}
			variants = append(variants, value)
			field.Variants = variants

			if _, ok := s.Attr("selected"); ok {
				field.Value = value
			}
		})

		fields[name] = field
	})

	return fields
}

// LotsByUser gets lots for provided userID. Key represents nodeID, value represents slice of offerIDs.
func (l *Lots) LotsByUser(ctx context.Context, userID int64) (map[string][]string, error) {
	const op = "Lots.LotsByUser"

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
func (l *Lots) extractLots(doc *goquery.Document) (map[string][]string, error) {
	const op = "lots.extractLots"

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

// UpdateLots updates lots for current account. Use [Lots.List] to get loaded lots.
// Returns [ErrAccountUnauthorized] if user id equals 0. Call [Funpay.Update] to update account info.
func (l *Lots) UpdateLots(ctx context.Context) error {
	const op = "Funpay.UpdateLots"

	id := l.fp.UserID()
	if id == 0 {
		return fmt.Errorf("%s: %w", op, ErrAccountUnauthorized)
	}

	lots, err := l.LotsByUser(context.Background(), id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	l.updateList(lots)

	return nil
}

// List returns loaded lots (see [Lots.Update]) in format nodeID: slice of offerIDs.
func (l *Lots) List() map[string][]string {
	l.mu.RLock()
	list := l.list
	l.mu.RUnlock()
	return list
}

func (l *Lots) updateList(list map[string][]string) {
	l.mu.Lock()
	l.list = list
	l.mu.Unlock()
}
