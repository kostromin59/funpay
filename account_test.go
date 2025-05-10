package funpay_test

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"net/http/httptest"
// 	"net/url"
// 	"strings"
// 	"testing"

// 	"github.com/kostromin59/funpay"
// )

// func TestAccount_Update(t *testing.T) {
// 	t.Run("successful update", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			http.SetCookie(w, &http.Cookie{Name: "test_cookie1", Value: "value1"})
// 			http.SetCookie(w, &http.Cookie{Name: "test_cookie2", Value: "value2"})

// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)

// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body data-app-data='{"userId":123,"csrf-token":"test-csrf","locale":"ru"}'>
// 						<div class="user-link-name">testuser</div>
// 						<div class="badge-balance">541 ₽</div>
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("valid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if err != nil {
// 			t.Fatalf("unexpected error: %v", err)
// 		}

// 		if account.UserID() != 123 {
// 			t.Errorf("expected userID 123, got %d", account.UserID())
// 		}

// 		if account.Username() != "testuser" {
// 			t.Errorf("expected username 'testuser', got %q", account.Username())
// 		}

// 		if account.Balance() != 541 {
// 			t.Errorf("expected balance 541, got %f", account.Balance())
// 		}

// 		if account.CSRFToken() != "test-csrf" {
// 			t.Errorf("expected csrf token 'test-csrf', got %q", account.CSRFToken())
// 		}

// 		if len(account.Cookies()) != 2 {
// 			t.Error("expected 2 cookies to be set")
// 		}

// 		if account.Locale() != "ru" {
// 			t.Errorf("expected Locale 'ru', got %q", account.Locale())
// 		}
// 	})

// 	t.Run("empty app data", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body>
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
// 			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
// 		}
// 	})

// 	t.Run("unauthorized", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body data-app-data='{"userId":0,"csrf-token":"test-csrf"}'>
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
// 			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
// 		}
// 	})

// 	t.Run("invalid json", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body data-app-data="invalid-json">
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if err == nil {
// 			t.Fatal("expected error, got nil")
// 		}
// 	})

// 	t.Run("invalid balance format", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body data-app-data="{"userId":123,"csrf-token":"test-csrf"}">
// 						<div class="user-link-name">testuser</div>
// 						<div class="badge-balance">invalid-balance</div>
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if err == nil {
// 			t.Fatal("expected error, got nil")
// 		}
// 	})

// 	t.Run("missing balance element", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 				<html>
// 					<body data-app-data='{"userId":123,"csrf-token":"test-csrf"}'>
// 						<div class="user-link-name">testuser</div>
// 					</body>
// 				</html>
// 			`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("valid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if err != nil {
// 			t.Fatalf("unexpected error: %v", err)
// 		}

// 		if account.Balance() != 0 {
// 			t.Errorf("expected balance 0 when element missing, got %f", account.Balance())
// 		}
// 	})

// 	t.Run("request error", func(t *testing.T) {
// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL("http://invalid-url")

// 		err := account.Update(t.Context())
// 		if err == nil {
// 			t.Fatal("expected error, got nil")
// 		}
// 	})

// 	t.Run("html parse error", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			w.WriteHeader(http.StatusOK)

// 			// Close connection
// 			hj, ok := w.(http.Hijacker)
// 			if !ok {
// 				t.Fatal("cannot hijack connection")
// 			}
// 			conn, _, _ := hj.Hijack()
// 			conn.Close()
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("valid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.Update(t.Context())
// 		if err == nil {
// 			t.Fatal("expected html parse error, got nil")
// 		}
// 	})
// }

// func TestAccount_UpdateLocale(t *testing.T) {
// 	t.Run("successful locale update", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if r.URL.Query().Get("setlocale") != string(funpay.LocaleEN) {
// 				t.Errorf("expected setlocale=%s, got %s", funpay.LocaleEN, r.URL.Query().Get("setlocale"))
// 			}

// 			w.WriteHeader(http.StatusOK)
// 			_, _ = w.Write([]byte(`
// 			<html>
// 					<body data-app-data='{"userId":123,"csrf-token":"test-csrf","locale":"en"}'>
// 							<div class="user-link-name">testuser</div>
// 							<div class="badge-balance">100 ₽</div>
// 					</body>
// 			</html>
// 	`))
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("valid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.UpdateLocale(t.Context(), funpay.LocaleEN)
// 		if err != nil {
// 			t.Fatalf("unexpected error: %v", err)
// 		}

// 		if account.Locale() != funpay.LocaleEN {
// 			t.Errorf("expected locale %s, got %s", funpay.LocaleEN, account.Locale())
// 		}
// 	})

// 	t.Run("unauthorized request", func(t *testing.T) {
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusForbidden)
// 		}))
// 		defer ts.Close()

// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL(ts.URL)

// 		err := account.UpdateLocale(t.Context(), funpay.LocaleEN)
// 		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
// 			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
// 		}
// 	})

// 	t.Run("invalid URL", func(t *testing.T) {
// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL("http://invalid-url")

// 		err := account.UpdateLocale(t.Context(), funpay.LocaleEN)
// 		if err == nil {
// 			t.Fatal("expected error, got nil")
// 		}
// 	})
// }

// func TestAccount_Request(t *testing.T) {
// 	t.Run("successful GET request", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if cookie, err := r.Cookie("golden_key"); err != nil || cookie.Value != "test_golden_key" {
// 				w.WriteHeader(http.StatusUnauthorized)
// 				return
// 			}

// 			if r.Header.Get("User-Agent") != "test_user_agent" {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(context.Background(), ts.URL, funpay.RequestWithUpdateAppData(false))
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("invalid URL", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		_, err := account.Request(context.Background(), "://invalid.url")
// 		if err == nil {
// 			t.Fatal("Expected error for invalid URL, got nil")
// 		}
// 	})

// 	t.Run("POST with body", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if r.Method != http.MethodPost {
// 				w.WriteHeader(http.StatusMethodNotAllowed)
// 				return
// 			}

// 			body, _ := io.ReadAll(r.Body)
// 			if string(body) != "test body" {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL,
// 			funpay.RequestWithMethod(http.MethodPost),
// 			funpay.RequestWithBody(strings.NewReader("test body")),
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("with cookies", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if cookie, err := r.Cookie("test_cookie"); err != nil || cookie.Value != "test_value" {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL,
// 			funpay.RequestWithCookies([]*http.Cookie{{Name: "test_cookie", Value: "test_value"}}),
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("with headers", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if r.Header.Get("X-Test") != "test-value" {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL,
// 			funpay.RequestWithHeaders(map[string]string{"X-Test": "test-value"}),
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("with proxy", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer proxyServer.Close()

// 		proxyURL, _ := url.Parse(proxyServer.URL)

// 		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			t.Error("Request should not reach target server directly")
// 			w.WriteHeader(http.StatusInternalServerError)
// 		}))
// 		defer targetServer.Close()

// 		account.SetProxy(proxyURL)
// 		resp, err := account.Request(
// 			context.Background(),
// 			targetServer.URL,
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("account cookies are sent", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if _, err := r.Cookie("account_cookie"); err != nil {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		account.SetCookies([]*http.Cookie{{Name: "account_cookie", Value: "test"}})
// 		resp, err := account.Request(context.Background(), ts.URL, funpay.RequestWithUpdateAppData(false))
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("cookies are updated from response", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			http.SetCookie(w, &http.Cookie{Name: "new_cookie", Value: "new_value"})
// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		_, err := account.Request(context.Background(), ts.URL, funpay.RequestWithUpdateAppData(false))
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		cookies := account.Cookies()
// 		found := false
// 		for _, cookie := range cookies {
// 			if cookie.Name == "new_cookie" && cookie.Value == "new_value" {
// 				found = true
// 				break
// 			}
// 		}

// 		if !found {
// 			t.Error("Account cookies were not updated from response")
// 		}
// 	})

// 	t.Run("locale handling - EN adds prefix", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if !strings.HasPrefix(r.URL.Path, "/en/") {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL+"/path",
// 			funpay.RequestWithLocale(funpay.LocaleEN),
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("locale handling - updateLocale adds query param", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if r.URL.Query().Get("setlocale") != "uk" {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL,
// 			funpay.RequestWithLocale(funpay.LocaleUK),
// 			funpay.RequestWithUpdateLocale(true),
// 			funpay.RequestWithUpdateAppData(false),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request() failed: %v", err)
// 		}

// 		if resp.StatusCode != http.StatusOK {
// 			t.Errorf("Expected status 200, got %d", resp.StatusCode)
// 		}
// 	})

// 	t.Run("unauthorized error", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusForbidden)
// 		}))
// 		defer ts.Close()

// 		_, err := account.Request(context.Background(), ts.URL)
// 		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
// 			t.Errorf("Expected ErrAccountUnauthorized, got %v", err)
// 		}
// 	})

// 	t.Run("rate limit error", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusTooManyRequests)
// 		}))
// 		defer ts.Close()

// 		_, err := account.Request(context.Background(), ts.URL)
// 		if !errors.Is(err, funpay.ErrTooManyRequests) {
// 			t.Errorf("Expected ErrTooManyRequests, got %v", err)
// 		}
// 	})

// 	t.Run("context cancellation", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusOK)
// 		}))
// 		defer ts.Close()

// 		ctx, cancel := context.WithCancel(context.Background())
// 		cancel()

// 		_, err := account.Request(ctx, ts.URL)
// 		if !errors.Is(err, context.Canceled) {
// 			t.Errorf("Expected context.Canceled, got %v", err)
// 		}
// 	})

// 	t.Run("update app data from response", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			if _, err := r.Cookie(funpay.GoldenKeyCookie); err != nil {
// 				w.WriteHeader(http.StatusUnauthorized)
// 				return
// 			}
// 			if r.Header.Get("User-Agent") != account.UserAgent() {
// 				w.WriteHeader(http.StatusBadRequest)
// 				return
// 			}

// 			appData := funpay.AppData{
// 				UserID:    12345,
// 				CSRFToken: "test_csrf_token",
// 				Locale:    funpay.LocaleEN,
// 			}
// 			data, _ := json.Marshal(appData)

// 			w.Header().Set("Content-Type", "text/html")
// 			fmt.Fprintf(w, `<html><body data-app-data='%s'></body></html>`, string(data))
// 		}))
// 		defer ts.Close()

// 		resp, err := account.Request(
// 			context.Background(),
// 			ts.URL,
// 			funpay.RequestWithUpdateAppData(true),
// 		)
// 		if err != nil {
// 			t.Fatalf("Request failed: %v", err)
// 		}
// 		defer resp.Body.Close()

// 		if account.UserID() != 12345 {
// 			t.Errorf("Expected UserID 12345, got %d", account.UserID())
// 		}

// 		if account.CSRFToken() != "test_csrf_token" {
// 			t.Errorf("Expected CSRF token 'test_csrf_token', got '%s'", account.CSRFToken())
// 		}

// 		if account.Locale() != funpay.LocaleEN {
// 			t.Errorf("Expected locale EN, got %v", account.Locale())
// 		}
// 	})

// 	t.Run("missing data-app-data attribute", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			_, _ = w.Write([]byte("<html><body></body></html>"))
// 		}))
// 		defer ts.Close()

// 		oldCSRF := account.CSRFToken()
// 		oldUserID := account.UserID()
// 		oldLocale := account.Locale()

// 		if account.CSRFToken() != oldCSRF {
// 			t.Error("CSRF token should not change on error")
// 		}

// 		if account.UserID() != oldUserID {
// 			t.Error("UserID should not change on error")
// 		}

// 		if account.Locale() != oldLocale {
// 			t.Error("Locale should not change on error")
// 		}
// 	})

// 	t.Run("invalid json in data-app-data", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			// Невалидный JSON
// 			fmt.Fprint(w, `<html><body data-app-data='{"userId": "not_a_number"}'></body></html>`)
// 		}))
// 		defer ts.Close()

// 		oldCSRF := account.CSRFToken()
// 		oldUserID := account.UserID()
// 		oldLocale := account.Locale()

// 		if account.CSRFToken() != oldCSRF {
// 			t.Error("CSRF token should not change on error")
// 		}

// 		if account.UserID() != oldUserID {
// 			t.Error("UserID should not change on error")
// 		}

// 		if account.Locale() != oldLocale {
// 			t.Error("Locale should not change on error")
// 		}
// 	})

// 	t.Run("empty userId in app data", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		testCSRF := "temp_csrf"
// 		testLocale := funpay.LocaleRU

// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Content-Type", "text/html")
// 			fmt.Fprintf(w, `<html><body data-app-data='{"userId":0,"csrf-token":"%s","locale":"%s"}'></body></html>`,
// 				testCSRF, testLocale)
// 		}))
// 		defer ts.Close()

// 		_, err := account.Request(t.Context(), ts.URL)
// 		if !errors.Is(err, funpay.ErrAccountUnauthorized) {
// 			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
// 		}

// 		if account.CSRFToken() != testCSRF {
// 			t.Errorf("CSRF token should update to %q, got %q", testCSRF, account.CSRFToken())
// 		}

// 		if account.Locale() != testLocale {
// 			t.Errorf("Locale should update to %v, got %v", testLocale, account.Locale())
// 		}

// 		if account.UserID() != 0 {
// 			t.Error("UserID should be 0 from test data")
// 		}
// 	})

// 	t.Run("request error", func(t *testing.T) {
// 		account := funpay.NewAccount("invalid_key", "test-agent")
// 		account.SetBaseURL("http://invalid-url")

// 		_, err := account.Request(t.Context(), "-", funpay.RequestWithMethod(":"))
// 		if err == nil {
// 			t.Fatal("expected error, got nil")
// 		}
// 	})

// 	t.Run("bad status code", func(t *testing.T) {
// 		account := funpay.NewAccount("test_golden_key", "test_user_agent")
// 		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			w.WriteHeader(http.StatusInternalServerError)
// 		}))
// 		defer ts.Close()

// 		_, err := account.Request(t.Context(), ts.URL)
// 		if !errors.Is(err, funpay.ErrBadStatusCode) {
// 			t.Fatalf("expected ErrAccountUnauthorized, got %v", err)
// 		}
// 	})
// }
