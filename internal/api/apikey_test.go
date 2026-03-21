package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
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

// TestGenerateAPIKey_Uniqueness generates many keys and checks they are all distinct.
func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	const n = 100
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		key := generateAPIKey()
		if seen[key] {
			t.Fatalf("duplicate key generated on iteration %d: %q", i, key)
		}
		seen[key] = true
	}
}

// TestGenerateAPIKey_Format verifies the key is "mb_" + 40 hex chars (20 random bytes).
func TestGenerateAPIKey_Format(t *testing.T) {
	key := generateAPIKey()
	// "mb_" prefix + 40 hex chars = 43 chars total
	if len(key) != 43 {
		t.Errorf("key length = %d, want 43", len(key))
	}
	hexPart := key[3:]
	matched, _ := regexp.MatchString("^[0-9a-f]{40}$", hexPart)
	if !matched {
		t.Errorf("key hex part is not 40 lowercase hex chars: %q", hexPart)
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

// TestAPIKeyPrefix_EdgeCases tests edge cases: empty string, exactly 10 chars, > 10 chars.
func TestAPIKeyPrefix_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"empty string", "", ""},
		{"single char", "a", "a"},
		{"exactly 10 chars", "0123456789", "0123456789"},
		{"11 chars truncates to 10", "0123456789X", "0123456789"},
		{"long key truncates to 10", "mb_abcdefghijklmnopqrstuvwxyz", "mb_abcdefg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiKeyPrefix(tt.key)
			if got != tt.want {
				t.Errorf("apiKeyPrefix(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
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

// TestHashAPIKey_SHA256Format verifies the hash is a 64-char lowercase hex string (SHA-256).
func TestHashAPIKey_SHA256Format(t *testing.T) {
	keys := []string{
		"mb_testapikey123",
		"mb_anotherkeyxyz",
		"",
		"a",
		"mb_0000000000000000000000000000000000000000",
	}
	hexPattern := regexp.MustCompile("^[0-9a-f]{64}$")
	for _, key := range keys {
		hash := hashAPIKey(key)
		if len(hash) != 64 {
			t.Errorf("hashAPIKey(%q) length = %d, want 64", key, len(hash))
		}
		if !hexPattern.MatchString(hash) {
			t.Errorf("hashAPIKey(%q) = %q, not a valid 64-char hex string", key, hash)
		}
	}
}

// TestHashAPIKey_EmptyInput verifies that hashing an empty string still produces a valid hash.
func TestHashAPIKey_EmptyInput(t *testing.T) {
	hash := hashAPIKey("")
	// SHA-256 of empty string is e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("hashAPIKey(\"\") = %q, want %q", hash, expected)
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

// TestAPIKeyMiddleware_InvalidAuthFormat tests Authorization headers that are not "Bearer <token>".
func TestAPIKeyMiddleware_InvalidAuthFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"Basic auth", "Basic dXNlcjpwYXNz"},
		{"bare token no scheme", "mb_testkey123"},
		{"empty bearer token", "Bearer "},
		{"token only", "some-random-token-no-scheme"},
		{"lowercase basic", "basic dXNlcjpwYXNz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Authorization", tt.header)

			APIKeyMiddleware(nil)(c)

			if c.IsAborted() {
				t.Errorf("should not abort for invalid auth format %q", tt.header)
			}
		})
	}
}

// TestAPIKeyMiddleware_NilQueries_ErrorResponse verifies the JSON error body on nil queries.
func TestAPIKeyMiddleware_NilQueries_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer mb_testkey123456789")

	APIKeyMiddleware(nil)(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, ok := resp["error"].(string)
	if !ok || errMsg == "" {
		t.Error("expected non-empty error field in response")
	}
	if errMsg != "API key authentication unavailable" {
		t.Errorf("error = %q, want %q", errMsg, "API key authentication unavailable")
	}
}
