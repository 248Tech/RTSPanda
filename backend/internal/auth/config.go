package auth

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCookieName      = "rtspanda_auth"
	defaultSessionTTLHours = 720 // 30 days
)

var (
	// ErrTokenRequired indicates auth is enabled but no token was configured.
	ErrTokenRequired = errors.New("AUTH_TOKEN is required when AUTH_ENABLED=true")
)

// Config controls API authentication behavior.
type Config struct {
	Enabled      bool
	Token        string
	CookieName   string
	CookieSecure bool
	SessionTTL   time.Duration
}

// LoadConfigFromEnv loads auth config from process env.
//
// Defaults:
//   - AUTH_ENABLED=true
//   - AUTH_COOKIE_NAME=rtspanda_auth
//   - AUTH_COOKIE_SECURE=false
//   - AUTH_SESSION_TTL_HOURS=720
func LoadConfigFromEnv() (Config, error) {
	enabled, err := envBool("AUTH_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Enabled: enabled,
	}
	if !cfg.Enabled {
		return cfg, nil
	}

	cfg.Token = strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	if cfg.Token == "" {
		return Config{}, ErrTokenRequired
	}

	cfg.CookieName = strings.TrimSpace(os.Getenv("AUTH_COOKIE_NAME"))
	if cfg.CookieName == "" {
		cfg.CookieName = defaultCookieName
	}

	cfg.CookieSecure, err = envBool("AUTH_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	ttlHours, err := envInt("AUTH_SESSION_TTL_HOURS", defaultSessionTTLHours)
	if err != nil {
		return Config{}, err
	}
	if ttlHours <= 0 {
		return Config{}, errors.New("AUTH_SESSION_TTL_HOURS must be greater than 0")
	}
	cfg.SessionTTL = time.Duration(ttlHours) * time.Hour

	return cfg, nil
}

func envBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New(key + " must be a boolean")
	}
	return v, nil
}

func envInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}
	return v, nil
}
