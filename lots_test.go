package funpay_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kostromin59/funpay"
)

func TestLots_SaveLot(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем только базовые требования
			if r.Method != http.MethodPost {
				t.Error("expected POST request")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		err := lots.SaveLot(context.Background(), funpay.LotFields{
			"offer_id": {Value: "123"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing offer_id", func(t *testing.T) {
		fp := funpay.New("test_key", "test_agent")
		lots := funpay.NewLots(fp)

		err := lots.SaveLot(context.Background(), funpay.LotFields{})
		if err == nil {
			t.Error("expected error when offer_id is missing")
		}
	})

	t.Run("request error handling", func(t *testing.T) {
		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL("http://unreachable-url")
		lots := funpay.NewLots(fp)

		err := lots.SaveLot(context.Background(), funpay.LotFields{
			"offer_id": {Value: "123"},
		})
		if err == nil {
			t.Error("expected error when request fails")
		}
	})
}

func TestLots_LotFields(t *testing.T) {
	t.Run("successful fields retrieval with offerID", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("offer") != "123" {
				t.Errorf("expected offer=123, got %s", r.URL.Query().Get("offer"))
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
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
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		fields, err := lots.LotFields(context.Background(), "", "123")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		expected := funpay.LotFields{
			"title":       {Value: "Test Title"},
			"active":      {Value: "on", Variants: []string{"on", ""}},
			"description": {Value: "Test Description"},
			"category":    {Value: "2", Variants: []string{"1", "2"}},
		}

		if !reflect.DeepEqual(fields, expected) {
			t.Errorf("expected %+v, got %+v", expected, fields)
		}
	})

	t.Run("successful fields retrieval with nodeID", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("node") != "456" {
				t.Errorf("expected node=456, got %s", r.URL.Query().Get("node"))
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
						<form>
							<input type="text" name="title" value="New Lot">
						</form>
					</body>
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		fields, err := lots.LotFields(context.Background(), "456", "")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		expected := funpay.LotFields{
			"title": {Value: "New Lot"},
		}

		if !reflect.DeepEqual(fields, expected) {
			t.Errorf("expected %+v, got %+v", expected, fields)
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL("http://invalid.url:12345")
		lots := funpay.NewLots(fp)

		_, err := lots.LotFields(context.Background(), "456", "")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer ts.Close()

		fp := funpay.New("invalid_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		_, err := lots.LotFields(context.Background(), "456", "")
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("empty form fields", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
						<form></form>
					</body>
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		fields, err := lots.LotFields(context.Background(), "456", "")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if len(fields) != 0 {
			t.Errorf("expected empty fields, got %+v", fields)
		}
	})

	t.Run("ignores CSRF token field", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
						<form>
							<input type="hidden" name="csrf_token" value="should-be-ignored">
							<input type="text" name="title" value="Test Title">
						</form>
					</body>
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		fields, err := lots.LotFields(context.Background(), "456", "")
		if err != nil {
			t.Fatalf("LotFields failed: %v", err)
		}

		if _, exists := fields["csrf-token"]; exists {
			t.Error("expected csrf-token field to be ignored")
		}
	})
}

func TestLots_LotsByUser(t *testing.T) {
	t.Run("successful lots retrieval", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/users/123/") {
				t.Errorf("expected path to end with /users/123/, got %s", r.URL.Path)
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
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
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		userLots, err := lots.LotsByUser(context.Background(), 123)
		if err != nil {
			t.Fatalf("LotsByUser failed: %v", err)
		}

		expected := map[string][]string{
			"game1": {"1", "2"},
			"game2": {"3"},
		}

		if !reflect.DeepEqual(userLots, expected) {
			t.Errorf("expected %v, got %v", expected, userLots)
		}
	})

	t.Run("invalid user URL", func(t *testing.T) {
		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL("http://invalid.url:12345")
		lots := funpay.NewLots(fp)

		_, err := lots.LotsByUser(context.Background(), 123)
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
	})

	t.Run("unauthorized request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer ts.Close()

		fp := funpay.New("invalid_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		_, err := lots.LotsByUser(context.Background(), 123)
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("no offers found", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html>
				<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'></body>
			</html>`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		userLots, err := lots.LotsByUser(context.Background(), 123)
		if err != nil {
			t.Fatalf("LotsByUser failed: %v", err)
		}

		if len(userLots) != 0 {
			t.Errorf("expected empty map, got %v", userLots)
		}
	})

	t.Run("malformed href in offer", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test","locale":"ru"}'>
						<div class="offer">
							<h3><a href=":invalid:url">Game 1</a></h3>
						</div>
					</body>
				</html>
			`)
		}))
		defer ts.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(ts.URL)
		lots := funpay.NewLots(fp)

		_, err := lots.LotsByUser(context.Background(), 123)
		if err == nil {
			t.Fatal("expected error for malformed URL, got nil")
		}
	})
}

func TestLots_UpdateLots(t *testing.T) {
	t.Run("successful lots update", func(t *testing.T) {
		accountTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
                <html>
                    <body data-app-data='{"userId":456,"csrf-token":"new-csrf","locale":"en"}'>
                        <div class="user-link-name">test_user</div>
                        <div class="badge-balance">100 ₽</div>
                    </body>
                </html>
            `)
		}))
		defer accountTS.Close()

		fp := funpay.New("test_key", "test_agent")
		fp.SetBaseURL(accountTS.URL)
		lots := funpay.NewLots(fp)

		if err := fp.Update(context.Background()); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		lotsTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/users/456/") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
                <html>
                     <body data-app-data='{"userId":456,"csrf-token":"new-csrf","locale":"en"}'>
                        <div class="offer">
                            <h3><a href="/games/game1/">Game 1</a></h3>
                            <a class="tc-item" href="/lots/lot1?id=1"></a>
                        </div>
                    </body>
                </html>
            `)
		}))
		defer lotsTS.Close()

		fp.SetBaseURL(lotsTS.URL)
		err := lots.UpdateLots(context.Background())
		if err != nil {
			t.Fatalf("UpdateLots failed: %v", err)
		}

		if len(lots.List()) == 0 {
			t.Error("expected lots to be updated")
		}
	})

	t.Run("unauthorized user", func(t *testing.T) {
		fp := funpay.New("test_key", "test_agent")
		lots := funpay.NewLots(fp)

		err := lots.UpdateLots(context.Background())
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})
}
