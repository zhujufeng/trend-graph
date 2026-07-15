// Package auth implements the single-admin boundary for the personal dashboard.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "trend_graph_session"

// SessionStore keeps the HTTP layer independent from the database implementation.
type SessionStore interface {
	Create(tokenHash string, expiresAt time.Time) error
	IsActive(tokenHash string, now time.Time) (bool, error)
	Delete(tokenHash string) error
}

type Service struct {
	password     []byte
	sessions     SessionStore
	ttl          time.Duration
	cookieSecure bool
	now          func() time.Time
}

func NewService(password string, sessions SessionStore, ttl time.Duration, cookieSecure bool) *Service {
	return &Service{
		password:     []byte(password),
		sessions:     sessions,
		ttl:          ttl,
		cookieSecure: cookieSecure,
		now:          time.Now,
	}
}

func (s *Service) Login(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(body.Password), s.password) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := newToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}
	expiresAt := s.now().Add(s.ttl)
	if err := s.sessions.Create(hashToken(token), expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not persist session"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(s.ttl.Seconds()),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	c.JSON(http.StatusOK, gin.H{"authenticated": true, "expiresAt": expiresAt})
}

func (s *Service) Logout(c *gin.Context) {
	if cookie, err := c.Request.Cookie(sessionCookieName); err == nil {
		_ = s.sessions.Delete(hashToken(cookie.Value))
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: s.cookieSecure, SameSite: http.SameSiteLaxMode,
	})
	c.Status(http.StatusNoContent)
}

func (s *Service) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Request.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		active, err := s.sessions.IsActive(hashToken(cookie.Value), s.now())
		if err != nil || !active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Next()
	}
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
