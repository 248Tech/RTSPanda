package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoadConfigFromEnv_RequiresTokenByDefault(t *testing.T) {
	t.Setenv("AUTH_ENABLED", "")
	t.Setenv("AUTH_TOKEN", "")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected error when AUTH_TOKEN is missing")
	}
}

func TestLoadConfigFromEnv_DisabledAllowsMissingToken(t *testing.T) {
	t.Setenv("AUTH_ENABLED", "false")
	t.Setenv("AUTH_TOKEN", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("expected auth to be disabled")
	}
}

func TestMiddlewareAllowsValidBearerToken(t *testing.T) {
	m := NewManager(Config{
		Enabled:    true,
		Token:      "secret-token",
		CookieName: "rtspanda_auth",
		SessionTTL: time.Hour,
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cameras", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()

	m.Middleware(next).ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
}

func TestMiddlewareRejectsMissingAuth(t *testing.T) {
	m := NewManager(Config{
		Enabled:    true,
		Token:      "secret-token",
		CookieName: "rtspanda_auth",
		SessionTTL: time.Hour,
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cameras", nil)
	rec := httptest.NewRecorder()

	m.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleLoginSetsSessionCookie(t *testing.T) {
	m := NewManager(Config{
		Enabled:      true,
		Token:        "secret-token",
		CookieName:   "rtspanda_auth",
		CookieSecure: true,
		SessionTTL:   24 * time.Hour,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"token":"secret-token"}`))
	rec := httptest.NewRecorder()

	m.HandleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	res := rec.Result()
	if len(res.Cookies()) == 0 {
		t.Fatalf("expected auth cookie to be set")
	}

	cookie := res.Cookies()[0]
	if cookie.Name != "rtspanda_auth" {
		t.Fatalf("unexpected cookie name %q", cookie.Name)
	}
	if cookie.Value != "secret-token" {
		t.Fatalf("unexpected cookie value %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatalf("cookie should be HttpOnly")
	}
	if !cookie.Secure {
		t.Fatalf("cookie should be Secure")
	}
}
