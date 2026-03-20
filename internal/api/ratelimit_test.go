package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/api"
)

func TestMetricsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newMockDBTX()
	router := setupRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty metrics body")
	}

	// Should contain Prometheus-format help lines
	if !contains(body, "brain_http_requests_total") {
		t.Error("expected brain_http_requests_total in metrics output")
	}
	if !contains(body, "brain_http_active_requests") {
		t.Error("expected brain_http_active_requests in metrics output")
	}
}

func TestMetrics_RecordsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	metrics := api.NewMetrics()

	r := gin.New()
	r.Use(metrics.Middleware())
	r.GET("/metrics", metrics.Handler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Make a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 from /test, got %d", w.Code)
	}

	// Now check metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !contains(body, `method="GET"`) {
		t.Error("expected method label in metrics")
	}
	if !contains(body, `path="/test"`) {
		t.Error("expected path label in metrics")
	}
	if !contains(body, `status="200"`) {
		t.Error("expected status label in metrics")
	}
}

func TestRateLimitMiddleware_NilRedis(t *testing.T) {
	// With nil Redis, rate limiting is disabled — requests should pass through
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(api.RateLimitMiddleware(nil, 1, 0))
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("request %d: expected 200 with nil Redis, got %d", i, w.Code)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
