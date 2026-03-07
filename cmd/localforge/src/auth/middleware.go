package auth

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const sessionContextKey = "localforge.auth.session"

type UnauthorizedMode int

const (
	UnauthorizedJSON UnauthorizedMode = iota
	UnauthorizedRedirect
)

func Middleware(cfg Config, store *SessionStore, mode UnauthorizedMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		token, err := c.Cookie(cfg.CookieName)
		if err == nil && token != "" {
			session, ok := store.Get(token)
			if ok {
				c.Set(sessionContextKey, session)
				c.Next()
				return
			}

			clearSessionCookie(c, cfg)
		}

		switch mode {
		case UnauthorizedRedirect:
			next := SafeNextPath(c.Request.URL.RequestURI())
			c.Redirect(http.StatusFound, "/login?next="+url.QueryEscape(next))
			c.Abort()
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		}
	}
}

func CurrentSession(c *gin.Context) (Session, bool) {
	value, ok := c.Get(sessionContextKey)
	if !ok {
		return Session{}, false
	}

	session, ok := value.(Session)
	return session, ok
}

func clearSessionCookie(c *gin.Context, cfg Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.ShouldUseSecureCookie(c.Request),
	})
}

func SafeNextPath(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}
