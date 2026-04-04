package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BillingConfig holds Stripe configuration.
type BillingConfig struct {
	SecretKey     string
	WebhookSecret string
	ProPriceID    string
	EntPriceID    string
}

// BillingHandler handles Stripe billing endpoints.
type BillingHandler struct {
	config    BillingConfig
	verifier  *WebhookVerifier
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(config BillingConfig, verifier *WebhookVerifier) *BillingHandler {
	h := &BillingHandler{
		config:   config,
		verifier: verifier,
	}

	// Register webhook secret for Stripe signature verification
	if config.WebhookSecret != "" {
		verifier.RegisterSecret("stripe", []byte(config.WebhookSecret))
	}

	return h
}

type createCheckoutRequest struct {
	Plan string `json:"plan" binding:"required,oneof=pro enterprise"`
}

// HandleCreateCheckout creates a Stripe Checkout Session.
func (h *BillingHandler) HandleCreateCheckout(c *gin.Context) {
	var req createCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan must be 'pro' or 'enterprise'"})
		return
	}

	tenantID := TenantFromContext(c)

	priceID := h.config.ProPriceID
	if req.Plan == "enterprise" {
		priceID = h.config.EntPriceID
	}

	// Build Stripe Checkout Session via API
	params := map[string]interface{}{
		"mode":        "subscription",
		"success_url": c.Request.Header.Get("Origin") + "/#/?checkout=success",
		"cancel_url":  c.Request.Header.Get("Origin") + "/#/?checkout=cancelled",
		"line_items": []map[string]interface{}{
			{
				"price":    priceID,
				"quantity": 1,
			},
		},
		"metadata": map[string]string{
			"tenant_id": tenantID,
			"plan":      req.Plan,
		},
	}

	body, err := json.Marshal(params)
	if err != nil {
		slog.Error("billing: marshal checkout params", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	_ = body // Stripe SDK would be used here; placeholder for now

	// Placeholder response — in production, call Stripe API
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"checkout_url": "https://checkout.stripe.com/placeholder",
			"session_id":   "cs_placeholder_" + tenantID,
		},
	})
}

// HandleStripeWebhook processes incoming Stripe webhook events.
func (h *BillingHandler) HandleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var event struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		slog.Info("billing: checkout completed", "event", event.Type)
		// FUTURE: Activate subscription for tenant (requires Stripe SDK + subscriptions table)
	case "customer.subscription.updated":
		slog.Info("billing: subscription updated", "event", event.Type)
		// FUTURE: Update tenant subscription status in DB
	case "customer.subscription.deleted":
		slog.Info("billing: subscription cancelled", "event", event.Type)
		// FUTURE: Downgrade tenant to free plan
	case "invoice.payment_failed":
		slog.Warn("billing: payment failed", "event", event.Type)
		// FUTURE: Notify tenant of payment failure via channel
	default:
		slog.Debug("billing: unhandled event", "type", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// handleBillingCheckout returns a gin handler that creates checkout sessions.
func handleBillingCheckout(cfg RouterConfig) gin.HandlerFunc {
	h := &BillingHandler{config: cfg.Billing}
	return h.HandleCreateCheckout
}

// handleBillingStatus returns a gin handler that checks billing status.
func handleBillingStatus(cfg RouterConfig) gin.HandlerFunc {
	h := &BillingHandler{config: cfg.Billing}
	return h.HandleBillingStatus
}

// HandleBillingStatus returns the current billing status for a tenant.
func (h *BillingHandler) HandleBillingStatus(c *gin.Context) {
	// Placeholder — in production, query Stripe for subscription status
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"plan":   "free",
			"status": "active",
		},
	})
}
