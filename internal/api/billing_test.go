package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func stringReader(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func TestBillingHandler_HandleCreateCheckout_InvalidPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &BillingHandler{config: BillingConfig{
		ProPriceID: "price_pro",
		EntPriceID: "price_ent",
	}}

	r := gin.New()
	r.POST("/billing/checkout", func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant")
		c.Set("role", "boss")
		c.Next()
	}, h.HandleCreateCheckout)

	req := httptest.NewRequest("POST", "/billing/checkout", stringReader(`{"plan":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBillingHandler_HandleCreateCheckout_Pro(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &BillingHandler{config: BillingConfig{
		ProPriceID: "price_pro",
		EntPriceID: "price_ent",
	}}

	r := gin.New()
	r.POST("/billing/checkout", func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-123")
		c.Set("role", "boss")
		c.Next()
	}, h.HandleCreateCheckout)

	req := httptest.NewRequest("POST", "/billing/checkout", stringReader(`{"plan":"pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "checkout_url") {
		t.Errorf("expected checkout_url in response, got %s", w.Body.String())
	}
}

func TestBillingHandler_HandleBillingStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &BillingHandler{config: BillingConfig{}}

	r := gin.New()
	r.GET("/billing/status", func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant")
		c.Next()
	}, h.HandleBillingStatus)

	req := httptest.NewRequest("GET", "/billing/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "free") {
		t.Errorf("expected 'free' plan in response, got %s", w.Body.String())
	}
}

func TestBillingHandler_HandleStripeWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &BillingHandler{config: BillingConfig{}}

	r := gin.New()
	r.POST("/webhooks/stripe", h.HandleStripeWebhook)

	body := `{"type":"checkout.session.completed","data":{}}`
	req := httptest.NewRequest("POST", "/webhooks/stripe", stringReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "received") {
		t.Errorf("expected 'received' in response, got %s", w.Body.String())
	}
}
