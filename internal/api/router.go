package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/roles"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
	"github.com/tonypk/ai-management-brain/internal/seats"
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
	MemoryEngine   *memory.MemoryEngine   // nil = memory disabled
	MemoryStore    *memory.MemoryStore    // nil = memory disabled
	ChannelRouter  *channel.Router         // nil = multi-channel disabled
	SlackAdapter   *channel.SlackAdapter   // nil = Slack disabled
	LarkAdapter    *channel.LarkAdapter    // nil = Lark disabled
	Scheduler      *scheduler.Scheduler    // nil = scheduler disabled
	SeatService    *seats.SeatService      // nil = seats disabled
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
		protected.PUT("/employees/:id/profile", RequireRole("boss"), handleUpdateProfile(cfg.Queries))

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

		// Seats (C-Suite management)
		protected.GET("/seats", handleListSeats(cfg.Queries))
		protected.POST("/seats", RequireRole("boss"), handleCreateSeat(cfg.Queries))
		protected.PUT("/seats/:id", RequireRole("boss"), handleUpdateSeat(cfg.Queries))
		protected.DELETE("/seats/:id", RequireRole("boss"), handleDeleteSeat(cfg.Queries))
		if cfg.SeatService != nil {
			protected.POST("/board/discuss", RequireRole("boss"), handleBoardDiscuss(cfg.SeatService))
		}
		protected.GET("/mentors", handleListMentorsWithDomain())

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

		// Memory routes (optional — requires memory engine)
		if cfg.MemoryStore != nil {
			memories := protected.Group("/memories")
			{
				memories.GET("", handleListMemories(cfg.MemoryStore))
				memories.GET("/:id", handleGetMemory(cfg.MemoryStore))
				memories.POST("/search", handleSearchMemories(cfg.MemoryEngine))
				memories.DELETE("/:id", RequireRole("boss"), handleDeleteMemory(cfg.MemoryStore))
				memories.POST("/consolidate", RequireRole("boss"), handleTriggerConsolidation(cfg.MemoryEngine))
			}

			// Employee profile via memory
			protected.GET("/employees/:id/profile", handleGetEmployeeProfile(cfg.MemoryStore))
		}

		// Admin routes (boss only)
		admin := protected.Group("/admin")
		admin.Use(RequireRole("boss"))
		{
			// Channels
			admin.GET("/channels", handleGetChannels(cfg.Queries, cfg.ChannelRouter))
			admin.PUT("/channels", handleUpdateChannels(cfg.Queries))
			admin.POST("/channels/test/:channel", handleTestChannel(cfg.ChannelRouter))

			// Employees with channels
			admin.GET("/employees", handleAdminListEmployees(cfg.Queries))
			admin.PUT("/employees/:id/channels", handleUpdateEmployeeChannels(cfg.Queries))
			admin.PUT("/employees/:id/preferred", handleUpdateEmployeePreferred(cfg.Queries))

			// Reports
			admin.GET("/reports", handleAdminListReports(cfg.Queries))
			admin.GET("/reports/stats", handleReportStats(cfg.Queries))

			// Group chats
			admin.GET("/groups", handleListGroups(cfg.Queries))
			admin.PUT("/groups/:id", handleUpdateGroup(cfg.Queries))
			admin.DELETE("/groups/:id", handleDeleteGroup(cfg.Queries))

			// Mentors (reuse existing mentorDescriptions from handlers.go)
			admin.GET("/mentors", handleListMentors())

			// Scheduler
			admin.GET("/scheduler", handleListSchedulerJobs(cfg.Scheduler))
			admin.PUT("/scheduler/:job/schedule", handleUpdateJobSchedule(cfg.Scheduler))
			admin.POST("/scheduler/:job/trigger", handleTriggerJob(cfg.Scheduler))

			// Memories (admin view — reuses existing handlers from memory_handlers.go)
			if cfg.MemoryStore != nil {
				admin.GET("/memories", handleListMemories(cfg.MemoryStore))
				admin.GET("/memories/stats", handleMemoryStats(cfg.MemoryStore))
				admin.DELETE("/memories/:id", handleDeleteMemory(cfg.MemoryStore))
				if cfg.MemoryEngine != nil {
					admin.POST("/memories/search", handleSearchMemories(cfg.MemoryEngine))
				}
			}
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

	// API Key-accessible endpoints for MCP server
	mcpAPI := v1.Group("")
	mcpAPI.Use(APIKeyMiddleware(cfg.Queries))
	mcpAPI.Use(AuthMiddleware(cfg.JWTSecret))
	{
		if cfg.SeatService != nil {
			mcpAPI.POST("/seats/chat", handleSeatChat(cfg.SeatService, cfg.Queries))
			mcpAPI.POST("/seats/board/discuss", handleBoardDiscuss(cfg.SeatService))
		}
		mcpAPI.GET("/seats/mentors", handleListMentorsWithDomain())
		mcpAPI.GET("/employees/profile/:name", handleEmployeeProfile(cfg.Queries))
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

	// Slack webhook (Events API — signature-verified by handler)
	if cfg.SlackAdapter != nil {
		v1.POST("/slack/events", cfg.SlackAdapter.HandleSlackEvent)
	}

	// Lark webhook (Event Subscription — token-verified by handler)
	if cfg.LarkAdapter != nil {
		v1.POST("/lark/events", cfg.LarkAdapter.HandleLarkEvent)
	}

	return r
}
