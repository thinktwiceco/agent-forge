package auth

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DefaultCookieName = "localforge_session"
	DefaultSessionTTL = 24 * time.Hour
)

type Config struct {
	Enabled      bool
	Username     string
	PasswordHash string
	CookieName   string
	SessionTTL   time.Duration
	SecureCookie bool
}

func LoadConfigFromEnv() Config {
	cfg := Config{
		Username:     strings.TrimSpace(os.Getenv("AUTH_USERNAME")),
		PasswordHash: strings.TrimSpace(os.Getenv("AUTH_PASSWORD_HASH")),
		CookieName:   DefaultCookieName,
		SessionTTL:   DefaultSessionTTL,
		SecureCookie: parseBoolEnv("AUTH_COOKIE_SECURE"),
	}

	if cookieName := strings.TrimSpace(os.Getenv("AUTH_COOKIE_NAME")); cookieName != "" {
		cfg.CookieName = cookieName
	}

	if ttlRaw := strings.TrimSpace(os.Getenv("AUTH_SESSION_TTL")); ttlRaw != "" {
		if ttl, err := time.ParseDuration(ttlRaw); err == nil && ttl > 0 {
			cfg.SessionTTL = ttl
		}
	}

	cfg.Enabled = cfg.Username != "" && cfg.PasswordHash != ""
	return cfg
}

func parseBoolEnv(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (c Config) CookieMaxAgeSeconds() int {
	return int(c.SessionTTL.Seconds())
}

func (c Config) ShouldUseSecureCookie(r *http.Request) bool {
	if c.SecureCookie {
		return true
	}
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
