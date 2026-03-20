package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WebhookVerifier verifies HMAC-SHA256 webhook signatures.
type WebhookVerifier struct {
	secrets map[string][]byte // provider → secret
}

// NewWebhookVerifier creates a new webhook verifier.
func NewWebhookVerifier() *WebhookVerifier {
	return &WebhookVerifier{
		secrets: make(map[string][]byte),
	}
}

// RegisterSecret registers a signing secret for a webhook provider.
func (w *WebhookVerifier) RegisterSecret(provider string, secret []byte) {
	w.secrets[provider] = secret
}

// VerifyMiddleware returns gin middleware that verifies webhook signatures.
// It reads the X-Signature-256 or X-Hub-Signature-256 header and compares
// against HMAC-SHA256 of the raw body.
func (w *WebhookVerifier) VerifyMiddleware(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret, ok := w.secrets[provider]
		if !ok || len(secret) == 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "webhook not configured"})
			return
		}

		// Read body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		// Get signature from headers (try multiple common header names)
		signature := c.GetHeader("X-Signature-256")
		if signature == "" {
			signature = c.GetHeader("X-Hub-Signature-256")
		}
		if signature == "" {
			signature = c.GetHeader("Stripe-Signature")
		}
		if signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing signature header"})
			return
		}

		// Compute HMAC-SHA256
		mac := hmac.New(sha256.New, secret)
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expected), []byte(signature)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		// Put body back for downstream handlers
		c.Request.Body = io.NopCloser(newBytesReader(body))
		c.Set("webhook_body", body)
		c.Next()
	}
}

// ComputeSignature computes the HMAC-SHA256 signature for a payload.
func ComputeSignature(payload, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// bytesReader wraps a byte slice for io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
