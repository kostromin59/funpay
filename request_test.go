package funpay_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kostromin59/funpay"
)

func TestRequest(t *testing.T) {
	account := funpay.NewAccount("test_key", "test_agent")

	t.Run("successful GET request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL).SetContext(t.Context())
		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("request creating error", func(t *testing.T) {
		req := funpay.NewRequest(account, "-").SetMethod(":").SetContext(t.Context())
		_, err := req.Do()
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("POST with body", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			body, _ := io.ReadAll(r.Body)
			if string(body) != "test body" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL).
			SetMethod(http.MethodPost).
			SetBody(strings.NewReader("test body")).
			SetContext(t.Context())

		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("custom cookies", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie("custom"); err != nil || cookie.Value != "value" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL).
			SetCookies([]*http.Cookie{{Name: "custom", Value: "value"}}).
			SetContext(t.Context())

		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("account cookies", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := r.Cookie("session"); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		account.SetCookies([]*http.Cookie{{Name: "session", Value: "test"}})
		req := funpay.NewRequest(account, ts.URL).SetContext(t.Context())

		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("custom headers", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Custom") != "value" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL).
			SetHeaders(map[string]string{"X-Custom": "value"}).
			SetContext(t.Context())

		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("request construction error", func(t *testing.T) {
		req := funpay.NewRequest(account, "://invalid.url").
			SetMethod("INVALID\nMETHOD").
			SetContext(t.Context())

		resp, err := req.Do()
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if resp != nil {
			t.Error("Expected nil response on construction error")
		}
	})

	t.Run("unauthorized error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL)
		resp, err := req.Do()
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Errorf("Expected ErrAccountUnauthorized, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
	})

	t.Run("rate limit error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL)
		resp, err := req.Do()
		if !errors.Is(err, funpay.ErrTooManyRequests) {
			t.Errorf("Expected ErrTooManyRequests, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
	})

	t.Run("bad status code", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL)
		resp, err := req.Do()
		if !errors.Is(err, funpay.ErrBadStatusCode) {
			t.Errorf("Expected ErrBadStatusCode, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		req := funpay.NewRequest(account, ts.URL).
			SetContext(ctx)

		resp, err := req.Do()
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got %v", err)
		}

		if resp != nil {
			t.Error("Expected nil response on context cancellation")
		}
	})

	t.Run("cookie update", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "new", Value: "value"})
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL)
		_, err := req.Do()
		if err != nil {
			t.Fatal(err)
		}

		cookies := account.Cookies()
		if len(cookies) != 1 || cookies[0].Name != "new" {
			t.Errorf("Expected new cookie, got %v", cookies)
		}
	})

	t.Run("proxy connection", func(t *testing.T) {
		proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != "target.example.com" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer proxyServer.Close()

		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Request should not reach target server directly")
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer targetServer.Close()

		proxyURL, err := url.Parse(proxyServer.URL)
		if err != nil {
			t.Fatalf("Failed to parse proxy URL: %v", err)
		}

		targetURL, _ := url.Parse(targetServer.URL)
		targetURL.Host = "target.example.com"

		req := funpay.NewRequest(account, targetURL.String()).
			SetContext(t.Context()).
			SetProxy(proxyURL)

		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("locale handling - EN adds prefix", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/en/") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL+"/path").SetLocale(funpay.LocaleEN)
		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("locale handling - EN adds prefix and empty path ends with /", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/en/") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL).SetLocale(funpay.LocaleEN)
		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("locale handling - RU doesn't add prefix", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/ru/") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL+"/path").SetLocale(funpay.LocaleRU)
		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("locale handling - updateLocale adds query param", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("setlocale") != "uk" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		req := funpay.NewRequest(account, ts.URL)
		req.SetLocale(funpay.LocaleUK).UpdateLocale(true)
		resp, err := req.Do()
		if err != nil {
			t.Fatalf("Do() failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}
