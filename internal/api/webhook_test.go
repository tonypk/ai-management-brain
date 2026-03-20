package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWebhookVerifier_ValidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := NewWebhookVerifier()
	secret := []byte("test-webhook-secret")
	verifier.RegisterSecret("stripe", secret)

	body := []byte(`{"type":"checkout.session.completed"}`)
	signature := ComputeSignature(body, secret)

	r := gin.New()
	r.POST("/webhooks/stripe", verifier.VerifyMiddleware("stripe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", signature)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookVerifier_InvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := NewWebhookVerifier()
	verifier.RegisterSecret("stripe", []byte("real-secret"))

	body := []byte(`{"type":"checkout.session.completed"}`)

	r := gin.New()
	r.POST("/webhooks/stripe", verifier.VerifyMiddleware("stripe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader(body))
	req.Header.Set("X-Signature-256", "sha256=invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebhookVerifier_MissingSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := NewWebhookVerifier()
	verifier.RegisterSecret("stripe", []byte("secret"))

	r := gin.New()
	r.POST("/webhooks/stripe", verifier.VerifyMiddleware("stripe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebhookVerifier_UnregisteredProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := NewWebhookVerifier()

	r := gin.New()
	r.POST("/webhooks/unknown", verifier.VerifyMiddleware("unknown"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/webhooks/unknown", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-Signature-256", "sha256=abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestComputeSignature(t *testing.T) {
	sig := ComputeSignature([]byte("hello"), []byte("secret"))
	if sig[:7] != "sha256=" {
		t.Errorf("signature should start with 'sha256=', got %q", sig[:7])
	}
	// Same input should produce same output
	sig2 := ComputeSignature([]byte("hello"), []byte("secret"))
	if sig != sig2 {
		t.Error("same input should produce same signature")
	}
	// Different input should produce different output
	sig3 := ComputeSignature([]byte("world"), []byte("secret"))
	if sig == sig3 {
		t.Error("different input should produce different signature")
	}
}
