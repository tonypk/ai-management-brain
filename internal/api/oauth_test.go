package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOAuthHandler_HandleGoogleClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOAuthHandler(nil, []byte("secret"), OAuthConfig{
		ClientID:     "test-client-id.apps.googleusercontent.com",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:3000/auth/callback",
	})

	r := gin.New()
	r.GET("/auth/google/client-id", handler.HandleGoogleClientID)

	req := httptest.NewRequest("GET", "/auth/google/client-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !contains(body, "test-client-id") {
		t.Errorf("expected client_id in response, got %s", body)
	}
}

func TestOAuthHandler_HandleGoogleCallback_NoCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewOAuthHandler(nil, []byte("secret"), OAuthConfig{})

	r := gin.New()
	r.POST("/auth/google", handler.HandleGoogleCallback)

	req := httptest.NewRequest("POST", "/auth/google", stringReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
