package main

import (
	"crypto/subtle"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	localauth "github.com/thinktwiceco/agent-forge/cmd/localforge/src/auth"
)

func (s *Server) handleLoginPage(c *gin.Context) {
	if !s.authConfig.Enabled {
		c.Redirect(http.StatusFound, "/")
		return
	}

	if token, err := c.Cookie(s.authConfig.CookieName); err == nil && token != "" {
		if _, ok := s.sessionStore.Get(token); ok {
			c.Redirect(http.StatusFound, requestedNextPath(c))
			return
		}
	}

	data, err := fs.ReadFile(s.staticFS, "login.html")
	if err != nil {
		c.String(http.StatusNotFound, "login.html not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (s *Server) handleAuthLogin(c *gin.Context) {
	if !s.authConfig.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "authentication is disabled"})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Username), []byte(s.authConfig.Username)) != 1 ||
		!localauth.VerifyPassword(req.Password, s.authConfig.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	token, session, err := s.sessionStore.Create(s.authConfig.Username, s.authConfig.SessionTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.authConfig.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   s.authConfig.CookieMaxAgeSeconds(),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.authConfig.ShouldUseSecureCookie(c.Request),
	})

	c.JSON(http.StatusOK, AuthStatusResponse{
		Enabled:       true,
		Authenticated: true,
		Username:      session.Username,
		Next:          requestedNextPath(c),
	})
}

func (s *Server) handleAuthLogout(c *gin.Context) {
	if token, err := c.Cookie(s.authConfig.CookieName); err == nil && token != "" {
		s.sessionStore.Delete(token)
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     s.authConfig.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.authConfig.ShouldUseSecureCookie(c.Request),
	})

	c.Status(http.StatusNoContent)
}

func (s *Server) handleAuthMe(c *gin.Context) {
	resp := AuthStatusResponse{
		Enabled: s.authConfig.Enabled,
		Next:    requestedNextPath(c),
	}
	if !s.authConfig.Enabled {
		c.JSON(http.StatusOK, resp)
		return
	}

	if token, err := c.Cookie(s.authConfig.CookieName); err == nil && token != "" {
		if session, ok := s.sessionStore.Get(token); ok {
			resp.Authenticated = true
			resp.Username = session.Username
		}
	}

	c.JSON(http.StatusOK, resp)
}

func requestedNextPath(c *gin.Context) string {
	return localauth.SafeNextPath(c.Query("next"))
}
