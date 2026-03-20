package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// RouterConfig holds dependencies for the API router.
type RouterConfig struct {
	Queries   *sqlc.Queries
	JWTSecret []byte
	Redis     *redis.Client // nil = no rate limiting
	OAuth     OAuthConfig   // Google OAuth config
	Billing   BillingConfig // Stripe billing config
}

// NewRouter creates the API router with public and protected routes.
func NewRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Metrics
	metrics := NewMetrics()
	r.Use(metrics.Middleware())
	r.GET("/metrics", metrics.Handler())

	// Rate limiting (60 req/min per IP)
	if cfg.Redis != nil {
		r.Use(RateLimitMiddleware(cfg.Redis, 60, time.Minute))
	}

	// Public routes
	v1 := r.Group("/api/v1")
	authHandler := NewAuthHandler(cfg.Queries, cfg.JWTSecret)
	v1.POST("/auth/login", authHandler.HandleLogin)
	v1.POST("/auth/register", authHandler.HandleRegister)

	// Google OAuth
	oauthHandler := NewOAuthHandler(cfg.Queries, cfg.JWTSecret, cfg.OAuth)
	v1.POST("/auth/google", oauthHandler.HandleGoogleCallback)
	v1.GET("/auth/google/client-id", oauthHandler.HandleGoogleClientID)

	// Protected routes
	protected := v1.Group("")
	protected.Use(AuthMiddleware(cfg.JWTSecret))
	{
		// Tenant
		protected.GET("/tenant", handleGetTenant(cfg.Queries))
		protected.PUT("/tenant", RequireRole("boss"), handleUpdateTenant(cfg.Queries))

		// Employees
		protected.GET("/employees", handleListEmployees(cfg.Queries))
		protected.POST("/employees", RequireRole("boss"), handleCreateEmployee(cfg.Queries))
		protected.GET("/employees/:id", handleGetEmployee(cfg.Queries))
		protected.PUT("/employees/:id/culture", RequireRole("boss"), handleUpdateEmployeeCulture(cfg.Queries))

		// Reports
		protected.GET("/reports", handleListReports(cfg.Queries))
		protected.GET("/reports/summary", handleGetSummary(cfg.Queries))

		// Mentor config
		protected.GET("/mentor", handleGetMentor(cfg.Queries))
		protected.PUT("/mentor", RequireRole("boss"), handleUpdateMentor(cfg.Queries))
		protected.PUT("/mentor/blend", RequireRole("boss"), handleUpdateBlend(cfg.Queries))

		// Dashboard stats
		protected.GET("/dashboard", handleDashboardStats(cfg.Queries))

		// Analytics (admin+)
		protected.GET("/analytics/overview", handleAnalyticsOverview(cfg.Queries))
		protected.GET("/analytics/activity", handleEmployeeActivity(cfg.Queries))

		// Billing
		protected.POST("/billing/checkout", handleBillingCheckout(cfg))
		protected.GET("/billing/status", handleBillingStatus(cfg))
	}

	// Webhook endpoints (signature-verified, no JWT)
	webhookVerifier := NewWebhookVerifier()
	billingHandler := NewBillingHandler(cfg.Billing, webhookVerifier)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/stripe", webhookVerifier.VerifyMiddleware("stripe"), billingHandler.HandleStripeWebhook)
	}

	return r
}
