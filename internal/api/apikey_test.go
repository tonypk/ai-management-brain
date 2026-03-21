package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateAPIKey(t *testing.T) {
	key := generateAPIKey()
	if len(key) < 40 {
		t.Errorf("key too short: %d", len(key))
	}
	if key[:3] != "mb_" {
		t.Errorf("key prefix wrong: %q", key[:3])
	}
	// Two keys should be different
	key2 := generateAPIKey()
	if key == key2 {
		t.Error("keys should be unique")
	}
}

func TestAPIKeyPrefix(t *testing.T) {
	tests := []struct{ key, want string }{
		{"mb_a1b2c3d4e5f6g7h8rest", "mb_a1b2c3d"},
		{"short", "short"},
		{"mb_exactten", "mb_exactte"},
	}
	for _, tt := range tests {
		got := apiKeyPrefix(tt.key)
		if got != tt.want {
			t.Errorf("apiKeyPrefix(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestHashAPIKey(t *testing.T) {
	key := "mb_testapikey123"
	hash := hashAPIKey(key)
	if hash == "" {
		t.Error("hash empty")
	}
	if hashAPIKey(key) != hash {
		t.Error("not deterministic")
	}
	if hashAPIKey("different") == hash {
		t.Error("different keys should have different hashes")
	}
}

func TestAPIKeyMiddleware_NoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	APIKeyMiddleware(nil)(c)

	if c.IsAborted() {
		t.Error("should not abort with no header")
	}
}

func TestAPIKeyMiddleware_NonAPIKeyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer eyJhbGciOiJI...")

	APIKeyMiddleware(nil)(c)

	if c.IsAborted() {
		t.Error("should not abort with JWT token")
	}
}

func TestAPIKeyMiddleware_APIKeyToken_NilQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer mb_testkey123")

	APIKeyMiddleware(nil)(c)

	if !c.IsAborted() {
		t.Error("should abort with nil queries")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
