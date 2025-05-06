package funpay_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kostromin59/funpay"
)

func TestAccount_Update(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "test_cookie1", Value: "value1"})
			http.SetCookie(w, &http.Cookie{Name: "test_cookie2", Value: "value2"})

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte(`
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test-csrf"}'>
						<div class="user-link-name">testuser</div>
						<div class="badge-balance">541 ₽</div>
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("valid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if account.UserID() != 123 {
			t.Errorf("expected userID 123, got %d", account.UserID())
		}

		if account.Username() != "testuser" {
			t.Errorf("expected username 'testuser', got %q", account.Username())
		}

		if account.Balance() != 541 {
			t.Errorf("expected balance 541, got %f", account.Balance())
		}

		if account.CSRFToken() != "test-csrf" {
			t.Errorf("expected csrf token 'test-csrf', got %q", account.CSRFToken())
		}

		if len(account.Cookies()) != 2 {
			t.Error("expected 2 cookies to be set")
		}
	})

	t.Run("empty app data", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
					<body>
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("invalid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
					<body data-app-data='{"userId":0,"csrf-token":"test-csrf"}'>
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("invalid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
					<body data-app-data="invalid-json">
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("invalid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid balance format", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
					<body data-app-data="{"userId":123,"csrf-token":"test-csrf"}">
						<div class="user-link-name">testuser</div>
						<div class="badge-balance">invalid-balance</div>
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("invalid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("missing balance element", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
					<body data-app-data='{"userId":123,"csrf-token":"test-csrf"}'>
						<div class="user-link-name">testuser</div>
					</body>
				</html>
			`))
		}))
		defer ts.Close()

		account := funpay.NewAccount("valid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if account.Balance() != 0 {
			t.Errorf("expected balance 0 when element missing, got %f", account.Balance())
		}
	})

	t.Run("request error", func(t *testing.T) {
		account := funpay.NewAccount("invalid_key", "test-agent")
		account.SetBaseURL("http://invalid-url")

		err := account.Update(t.Context())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("html parse error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)

			// Close connection
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("cannot hijack connection")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
		}))
		defer ts.Close()

		account := funpay.NewAccount("valid_key", "test-agent")
		account.SetBaseURL(ts.URL)

		err := account.Update(t.Context())
		if err == nil {
			t.Fatal("expected html parse error, got nil")
		}
	})

	t.Run("with proxy", func(t *testing.T) {
		proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Via") != "test-proxy" {
				t.Error("request didn't go through proxy")
			}
		}))
		defer proxyServer.Close()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Via", "test-proxy")

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
            <html>
                <body data-app-data='{"userId":123,"csrf-token":"test-csrf"}'>
                    <div class="user-link-name">testuser</div>
                    <div class="badge-balance">100 ₽</div>
                </body>
            </html>
        `))
		}))
		defer ts.Close()

		account := funpay.NewAccount("proxy_key", "test-agent")
		account.SetBaseURL(ts.URL)

		proxyURL, err := url.Parse(proxyServer.URL)
		if err != nil {
			t.Fatalf("failed to parse proxy URL: %v", err)
		}

		account.SetProxy(proxyURL)

		err = account.Update(t.Context())
		if err != nil {
			t.Fatalf("unexpected error with proxy: %v", err)
		}

		if account.UserID() != 123 {
			t.Errorf("expected userID 123 with proxy, got %d", account.UserID())
		}
	})
}
