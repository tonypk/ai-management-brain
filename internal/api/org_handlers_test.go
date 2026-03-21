package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandleStartWizard_NilWizard(t *testing.T) {
	r := gin.New()
	r.POST("/org/wizard/start", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Set("role", "boss")
		c.Next()
	}, handleStartWizard(nil, nil))

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"mentor_id": "musk"}`)
	req, _ := http.NewRequest("POST", "/org/wizard/start", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleStartWizard_MissingMentorID(t *testing.T) {
	r := gin.New()
	r.POST("/org/wizard/start", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Set("role", "boss")
		c.Next()
	}, handleStartWizard(nil, nil))

	w := httptest.NewRecorder()
	body := strings.NewReader(`{}`)
	req, _ := http.NewRequest("POST", "/org/wizard/start", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// nil wizard check comes before validation, so 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleWizardAnswer_NilWizard(t *testing.T) {
	r := gin.New()
	r.POST("/org/wizard/answer", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Set("role", "boss")
		c.Next()
	}, handleWizardAnswer(nil, nil))

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"answer": "test"}`)
	req, _ := http.NewRequest("POST", "/org/wizard/answer", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleGetPlan_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.GET("/org/plan", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleGetPlan(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/org/plan", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleUpdatePlan_NilEngine(t *testing.T) {
	r := gin.New()
	r.PUT("/org/plan", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Set("role", "boss")
		c.Next()
	}, handleUpdatePlan(nil, nil))

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"feedback": "remove C-level"}`)
	req, _ := http.NewRequest("PUT", "/org/plan", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleUpdatePlan_MissingFeedback(t *testing.T) {
	r := gin.New()
	r.PUT("/org/plan", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "00000000-0000-0000-0000-000000000001")
		c.Set("role", "boss")
		c.Next()
	}, handleUpdatePlan(nil, nil))

	w := httptest.NewRecorder()
	body := strings.NewReader(`{}`)
	req, _ := http.NewRequest("PUT", "/org/plan", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// nil engine check comes before validation, so 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandleActivatePlan_InvalidTenant(t *testing.T) {
	r := gin.New()
	r.POST("/org/plan/activate", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Set("tenant_id", "invalid-uuid")
		c.Set("role", "boss")
		c.Next()
	}, handleActivatePlan(nil))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/org/plan/activate", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
