package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleCheckin_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/checkin", nil)

	h := NewOpenClawActionHandler(nil)
	h.HandleCheckin(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "invalid tenant")
}

func TestHandleChase_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/chase", nil)

	h := NewOpenClawActionHandler(nil)
	h.HandleChase(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "invalid tenant")
}

func TestHandleSummary_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/summary", nil)

	h := NewOpenClawActionHandler(nil)
	h.HandleSummary(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "invalid tenant")
}

func TestHandleMessage_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"employee_name":"John","message":"hello"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/message", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h := NewOpenClawActionHandler(nil)
	h.HandleMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "invalid tenant")
}

func TestHandleMessage_NoBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/message", nil)
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

	h := NewOpenClawActionHandler(nil)
	h.HandleMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "employee_name and message are required")
}

func TestHandleMessage_MissingMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"employee_name":"John"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/message", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

	h := NewOpenClawActionHandler(nil)
	h.HandleMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	assertErrorMessage(t, w, "employee_name and message are required")
}

func TestHandleMessage_MissingEmployeeName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"message":"hello"}`
	c.Request, _ = http.NewRequest("POST", "/api/v1/openclaw/message", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")

	h := NewOpenClawActionHandler(nil)
	h.HandleMessage(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCheckin_EmptyBody_NilService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		h := NewOpenClawActionHandler(nil)
		h.HandleCheckin(c)
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// nil service should panic → recovered to 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil service panic recovered)", w.Code)
	}
}

func TestHandleChase_EmptyBody_NilService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		h := NewOpenClawActionHandler(nil)
		h.HandleChase(c)
	})

	req, _ := http.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil service panic recovered)", w.Code)
	}
}

func TestHandleSummary_NilService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/test", func(c *gin.Context) {
		c.Set("tenant_id", "550e8400-e29b-41d4-a716-446655440000")
		h := NewOpenClawActionHandler(nil)
		h.HandleSummary(c)
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (nil service panic recovered)", w.Code)
	}
}

func assertErrorMessage(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	errMsg, _ := resp["error"].(string)
	if errMsg != expected {
		t.Errorf("error = %q, want %q", errMsg, expected)
	}
}
