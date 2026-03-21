package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/roles"
)

// RouterConfig holds dependencies for the API router.
type RouterConfig struct {
	Queries     *sqlc.Queries
	JWTSecret   []byte
	Redis       *redis.Client    // nil = no rate limiting
	OAuth       OAuthConfig      // Google OAuth config
	Billing     BillingConfig    // Stripe billing config
	OrgWizard   *brain.OrgWizard // nil = org features disabled
	OrgEngine   *brain.OrgEngine // nil = org features disabled
	RoleManager    *roles.Manager          // nil = AI roles disabled
	SignalAdapter  *channel.SignalAdapter  // nil = Signal disabled
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

		// API Key management (JWT-authenticated only)
		protected.POST("/auth/api-keys", handleCreateAPIKey(cfg.Queries))
		protected.GET("/auth/api-keys", handleListAPIKeys(cfg.Queries))
		protected.DELETE("/auth/api-keys/:id", handleRevokeAPIKey(cfg.Queries))

		// Organization architecture (boss only)
		org := protected.Group("/org")
		org.Use(RequireRole("boss"))
		{
			org.POST("/wizard/start", handleStartWizard(cfg.Queries, cfg.OrgWizard))
			org.POST("/wizard/answer", handleWizardAnswer(cfg.Queries, cfg.OrgWizard))
			org.GET("/plan", handleGetPlan(cfg.Queries))
			org.PUT("/plan", handleUpdatePlan(cfg.Queries, cfg.OrgEngine))
			org.POST("/plan/activate", handleActivatePlan(cfg.Queries, cfg.RoleManager))

			// AI Roles
			org.GET("/roles", handleListAIRoles(cfg.Queries))
			org.GET("/suggestions", handleListSuggestions(cfg.Queries))
			org.POST("/suggestions/:id/approve", handleApproveSuggestion(cfg.Queries))
			org.POST("/suggestions/:id/reject", handleRejectSuggestion(cfg.Queries))
		}
	}

	// OpenClaw endpoints (API Key or JWT authenticated)
	openclaw := v1.Group("/openclaw")
	openclaw.Use(APIKeyMiddleware(cfg.Queries))
	openclaw.Use(AuthMiddleware(cfg.JWTSecret))
	{
		openclaw.GET("/status", handleOpenClawStatus(cfg.Queries))
		openclaw.POST("/command", handleOpenClawCommand(cfg.Queries))
		openclaw.GET("/report", handleOpenClawReport(cfg.Queries))
		openclaw.GET("/alerts", handleOpenClawAlerts(cfg.Queries))
	}

	// Webhook endpoints (signature-verified, no JWT)
	webhookVerifier := NewWebhookVerifier()
	billingHandler := NewBillingHandler(cfg.Billing, webhookVerifier)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/stripe", webhookVerifier.VerifyMiddleware("stripe"), billingHandler.HandleStripeWebhook)
	}

	// Signal webhook (no auth — called by signal-cli container on internal network)
	if cfg.SignalAdapter != nil {
		v1.POST("/signal/webhook", cfg.SignalAdapter.HandleWebhook)
	}

	return r
}
