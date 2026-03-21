package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleListAIRoles_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.GET("/org/roles", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleListAIRoles(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/org/roles", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleListSuggestions_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.GET("/org/suggestions", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleListSuggestions(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/org/suggestions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleApproveSuggestion_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.POST("/org/suggestions/:id/approve", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleApproveSuggestion(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/org/suggestions/some-id/approve", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleRejectSuggestion_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.POST("/org/suggestions/:id/reject", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleRejectSuggestion(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/org/suggestions/some-id/reject", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
