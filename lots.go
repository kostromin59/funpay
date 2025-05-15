package funpay

import (
	"fmt"
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

type lots struct {
	list map[string][]string
	mu   sync.RWMutex
}

func newLots() *lots {
	return &lots{}
}

// List returns loaded lots (see [Funpay.UpdateLots]) in format nodeID: slice of offerIDs.
func (l *lots) List() map[string][]string {
	l.mu.RLock()
	list := l.list
	l.mu.RUnlock()
	return list
}

func (l *lots) updateList(list map[string][]string) {
	l.mu.Lock()
	l.list = list
	l.mu.Unlock()
}

func (l *lots) extractLots(doc *goquery.Document) (map[string][]string, error) {
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

func (l *lots) extractFields(doc *goquery.Document) LotFields {
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
