package lots_test

import (
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/kostromin59/funpay"
	"github.com/kostromin59/funpay/lots"
	"github.com/kostromin59/funpay/mocks"
	"go.uber.org/mock/gomock"
)

func TestLots_Save(t *testing.T) {
	t.Parallel()
	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		saveLotFields := lots.Fields{
			"offer_id": lots.Field{Value: "123"},
		}

		expectedBody := url.Values{}
		for name, v := range saveLotFields {
			expectedBody.Set(string(name), v.Value)
		}
		expectedBody.Set(funpay.FormCSRFToken, "csrf")
		expectedBody.Set("location", "trade")

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().CSRFToken().Times(1).Return("csrf")
		fp.EXPECT().Request(
			t.Context(),
			"https://funpay.com/lots/offerSave",
			gomock.Any(),
		).Times(1).Return(nil, nil)

		err := fpLots.Save(t.Context(), saveLotFields)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("request error handling", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().CSRFToken().Times(1).Return("csrf")
		fp.EXPECT().Request(
			t.Context(),
			"https://funpay.com/lots/offerSave",
			gomock.Any(),
		).Times(1).Return(nil, errors.New("request error"))

		err := fpLots.Save(t.Context(), lots.Fields{
			"offer_id": lots.Field{Value: "123"},
		})
		if err == nil {
			t.Error("expected error when request fails")
		}
	})
}

func TestLots_FieldsByOfferID(t *testing.T) {
	t.Run("successful fields retrieval with offerID", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form>
					<input type="text" name="title" value="Test Title">
					<input type="checkbox" name="active" checked>
					<textarea name="description">Test Description</textarea>
					<select name="category">
						<option value="1">Category 1</option>
						<option value="2" selected>Category 2</option>
					</select>
				</form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?offer=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByOfferID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		expected := lots.Fields{
			"title":       lots.Field{Value: "Test Title"},
			"active":      lots.Field{Value: "on", Variants: []string{"on"}},
			"description": lots.Field{Value: "Test Description"},
			"category":    lots.Field{Value: "2", Variants: []string{"1", "2"}},
		}

		if !reflect.DeepEqual(fields, expected) {
			t.Errorf("expected %+v, got %+v", expected, fields)
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return(":not a url")

		_, err := fpLots.FieldsByOfferID(t.Context(), "something")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?offer=something",
		).Times(1).Return(nil, funpay.ErrAccountUnauthorized)

		_, err := fpLots.FieldsByOfferID(t.Context(), "something")
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("empty form fields", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form></form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?offer=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByOfferID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if len(fields) != 0 {
			t.Errorf("expected empty fields, got %+v", fields)
		}
	})

	t.Run("ignores CSRF token field", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form>
					<input type="hidden" name="csrf_token" value="should-be-ignored">
					<input type="text" name="title" value="Test Title">
				</form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?offer=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByOfferID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if _, exists := fields["csrf-token"]; exists {
			t.Error("expected csrf-token field to be ignored")
		}
	})
}

func TestLots_FieldsByNodeID(t *testing.T) {
	t.Run("successful fields retrieval with nodeID", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form>
					<input type="text" name="title" value="New Lot">
				</form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?node=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByNodeID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		expected := lots.Fields{
			"title": lots.Field{Value: "New Lot"},
		}

		if !reflect.DeepEqual(fields, expected) {
			t.Errorf("expected %+v, got %+v", expected, fields)
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return(":not a url")

		_, err := fpLots.FieldsByNodeID(t.Context(), "something")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?node=something",
		).Times(1).Return(nil, funpay.ErrAccountUnauthorized)

		_, err := fpLots.FieldsByNodeID(t.Context(), "something")
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("empty form fields", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form></form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?node=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByNodeID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if len(fields) != 0 {
			t.Errorf("expected empty fields, got %+v", fields)
		}
	})

	t.Run("ignores CSRF token field", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<form>
					<input type="hidden" name="csrf_token" value="should-be-ignored">
					<input type="text" name="title" value="Test Title">
				</form>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/lots/offerEdit?node=something",
		).Times(1).Return(doc, nil)

		fields, err := fpLots.FieldsByNodeID(t.Context(), "something")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if _, exists := fields["csrf-token"]; exists {
			t.Error("expected csrf-token field to be ignored")
		}
	})
}

func TestLots_ByUser(t *testing.T) {
	t.Parallel()
	t.Run("successful lots retrieval", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
				<div class="offer">
					<h3><a href="/games/game1/">Game 1</a></h3>
					<a class="tc-item" href="/lots/lot1?id=1"></a>
					<a class="tc-item" href="/lots/lot2?id=2"></a>
				</div>
				<div class="offer">
					<h3><a href="/games/game2/">Game 2</a></h3>
					<a class="tc-item" href="/lots/lot3?id=3"></a>
				</div>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/users/123/",
		).Times(1).Return(doc, nil)

		userLots, err := fpLots.ByUser(t.Context(), 123)
		if err != nil {
			t.Fatalf("LotsByUser failed: %v", err)
		}

		expected := map[lots.NodeID][]lots.OfferID{
			"game1": {"1", "2"},
			"game2": {"3"},
		}

		if !reflect.DeepEqual(userLots, expected) {
			t.Errorf("expected %v, got %v", expected, userLots)
		}
	})

	t.Run("invalid user URL", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return(":not a url")

		_, err := fpLots.ByUser(t.Context(), 123)
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/users/123/",
		).Times(1).Return(nil, funpay.ErrAccountUnauthorized)

		_, err := fpLots.ByUser(t.Context(), 123)
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("no offers found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
				<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'></body>
			</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/users/123/",
		).Times(1).Return(doc, nil)

		userLots, err := fpLots.ByUser(t.Context(), 123)
		if err != nil {
			t.Fatalf("LotsByUser failed: %v", err)
		}

		if len(userLots) != 0 {
			t.Errorf("expected empty map, got %v", userLots)
		}
	})

	t.Run("invalid href in offer", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
					<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
						<div class="offer">
							<h3><a href=":invalid:url">Game 1</a></h3>
						</div>
					</body>
				</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/users/123/",
		).Times(1).Return(doc, nil)

		_, err = fpLots.ByUser(t.Context(), 123)
		if err == nil {
			t.Fatal("expected error for malformed URL, got nil")
		}
	})
}

func TestLots_UpdateLots(t *testing.T) {
	t.Parallel()
	t.Run("successful lots update", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<html>
			<body data-app-data='{"userId":456,"csrf-token":"new-csrf","locale":"en"}'>
				<div class="offer">
					<h3><a href="/games/game1/">Game 1</a></h3>
					<a class="tc-item" href="/lots/lot1?id=1"></a>
				</div>
			</body>
		</html>`))
		if err != nil {
			t.Fatal("invalid doc provided")
		}

		fp.EXPECT().UserID().Times(1).Return(int64(456))
		fp.EXPECT().BaseURL().Times(1).Return("https://funpay.com")
		fp.EXPECT().RequestHTML(
			t.Context(),
			"https://funpay.com/users/456/",
		).Times(1).Return(doc, nil)

		err = fpLots.Update(t.Context())
		if err != nil {
			t.Fatalf("UpdateLots failed: %v", err)
		}

		if len(fpLots.List()) == 0 {
			t.Error("expected lots to be updated")
		}
	})

	t.Run("unauthorized user", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fp := mocks.NewMockFunpay(ctrl)
		fpLots := lots.New(fp)

		fp.EXPECT().UserID().Times(1).Return(int64(0))

		err := fpLots.Update(t.Context())
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})
}
