package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type memorySessions struct {
	sessions map[string]time.Time
}

func newMemorySessions() *memorySessions {
	return &memorySessions{sessions: make(map[string]time.Time)}
}

func (m *memorySessions) Create(tokenHash string, expiresAt time.Time) error {
	m.sessions[tokenHash] = expiresAt
	return nil
}

func (m *memorySessions) IsActive(tokenHash string, now time.Time) (bool, error) {
	expiresAt, ok := m.sessions[tokenHash]
	return ok && expiresAt.After(now), nil
}

func (m *memorySessions) Delete(tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func TestLoginProtectsAndRevokesSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewService("correct horse battery staple", newMemorySessions(), time.Hour, false)
	r := gin.New()
	r.POST("/login", svc.Login)
	r.POST("/logout", svc.Require(), svc.Logout)
	r.GET("/private", svc.Require(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	wrong := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"password":"wrong"}`))
	wrong.Header.Set("Content-Type", "application/json")
	wrongResult := httptest.NewRecorder()
	r.ServeHTTP(wrongResult, wrong)
	if wrongResult.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want %d", wrongResult.Code, http.StatusUnauthorized)
	}

	login := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"password":"correct horse battery staple"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	r.ServeHTTP(loginResult, login)
	if loginResult.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResult.Code, loginResult.Body.String())
	}
	cookies := loginResult.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatalf("login must issue one HttpOnly cookie, got %#v", cookies)
	}

	private := httptest.NewRequest(http.MethodGet, "/private", nil)
	private.AddCookie(cookies[0])
	privateResult := httptest.NewRecorder()
	r.ServeHTTP(privateResult, private)
	if privateResult.Code != http.StatusNoContent {
		t.Fatalf("authenticated request status = %d", privateResult.Code)
	}

	logout := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logout.AddCookie(cookies[0])
	logoutResult := httptest.NewRecorder()
	r.ServeHTTP(logoutResult, logout)
	if logoutResult.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logoutResult.Code)
	}

	privateAfterLogout := httptest.NewRequest(http.MethodGet, "/private", nil)
	privateAfterLogout.AddCookie(cookies[0])
	privateAfterLogoutResult := httptest.NewRecorder()
	r.ServeHTTP(privateAfterLogoutResult, privateAfterLogout)
	if privateAfterLogoutResult.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d, want %d", privateAfterLogoutResult.Code, http.StatusUnauthorized)
	}
}
