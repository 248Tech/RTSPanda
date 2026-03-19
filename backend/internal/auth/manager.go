package auth

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Manager validates tokens and exposes auth-related HTTP handlers.
type Manager struct {
	cfg Config
}

// NewManager returns a configured auth manager.
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

// Enabled returns true when auth checks are active.
func (m *Manager) Enabled() bool {
	return m.cfg.Enabled
}

// Mode returns the configured auth mode.
func (m *Manager) Mode() string {
	if !m.cfg.Enabled {
		return "none"
	}
	return "bearer_token"
}

// Middleware enforces auth on the wrapped handler.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	if !m.cfg.Enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.IsRequestAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeJSONError(w, http.StatusUnauthorized, "missing or invalid auth token")
	})
}

// IsRequestAuthenticated returns true when the request carries a valid token.
func (m *Manager) IsRequestAuthenticated(r *http.Request) bool {
	if !m.cfg.Enabled {
		return true
	}

	if bearerToken := extractBearerToken(r.Header.Get("Authorization")); m.isValidToken(bearerToken) {
		return true
	}

	cookie, err := r.Cookie(m.cfg.CookieName)
	if err != nil {
		return false
	}
	return m.isValidToken(cookie.Value)
}

// HandleConfig exposes auth mode metadata used by the frontend bootstrap flow.
func (m *Manager) HandleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": m.cfg.Enabled,
		"mode":    m.Mode(),
	})
}

// HandleSession checks whether the incoming request has an active auth session.
func (m *Manager) HandleSession(w http.ResponseWriter, r *http.Request) {
	if !m.cfg.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":       false,
			"authenticated": true,
		})
		return
	}

	if !m.IsRequestAuthenticated(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"enabled":       true,
			"authenticated": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":       true,
		"authenticated": true,
	})
}

// HandleLogin validates a token and sets an HttpOnly auth cookie.
func (m *Manager) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if !m.cfg.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":       false,
			"authenticated": true,
		})
		return
	}

	type loginRequest struct {
		Token string `json:"token"`
	}
	var in loginRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if !m.isValidToken(in.Token) {
		writeJSONError(w, http.StatusUnauthorized, "invalid auth token")
		return
	}

	expiry := time.Now().UTC().Add(m.cfg.SessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    m.cfg.Token,
		Path:     "/",
		Expires:  expiry,
		MaxAge:   int(m.cfg.SessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   m.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":       true,
		"authenticated": true,
		"expires_at":    expiry.Format(time.RFC3339),
	})
}

// HandleLogout clears the auth cookie.
func (m *Manager) HandleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   m.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": false,
	})
}

func (m *Manager) isValidToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || !m.cfg.Enabled {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(m.cfg.Token)) == 1
}

func extractBearerToken(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
