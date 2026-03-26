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
	"github.com/tonypk/ai-management-brain/internal/service"
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
	ActionService  *service.ActionService  // nil = action endpoints disabled
	HalaOSMapper   halaosMapper            // nil = HalaOS dispatch disabled (wired in Task 14)
	Recommender    *brain.Recommender     // nil = recommendation engine disabled
	Dispatcher     *brain.Dispatcher      // nil = recommendation dispatch disabled
}

// NewRouter creates the API router with public and protected routes.
func NewRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Metrics
	metrics := NewMetrics()
	r.Use(metrics.Middleware())
	r.GET("/__metrics", metrics.Handler())

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
			org.POST("/setup", handleSetupOrg(cfg.Queries, cfg.OrgEngine))
			org.GET("/plan", handleGetPlan(cfg.Queries))
			org.PUT("/plan", handleUpdatePlan(cfg.Queries, cfg.OrgEngine))
			org.POST("/plan/activate", handleActivatePlan(cfg.Queries, cfg.RoleManager))

			// AI Roles
			org.GET("/roles", handleListAIRoles(cfg.Queries))
			org.GET("/suggestions", handleListSuggestions(cfg.Queries))
			org.POST("/suggestions/:id/approve", handleApproveSuggestion(cfg.Queries))
			org.POST("/suggestions/:id/reject", handleRejectSuggestion(cfg.Queries))
		}

		// Goals/OKR
		goals := protected.Group("/goals")
		goals.Use(RequireRole("boss"))
		{
			goals.GET("", handleListGoals(cfg.Queries))
			goals.POST("", handleCreateGoal(cfg.Queries))
			goals.PUT("/:id", handleUpdateGoal(cfg.Queries))
			goals.DELETE("/:id", handleDeleteGoal(cfg.Queries))
			goals.POST("/:id/key-results", handleCreateKeyResult(cfg.Queries))
			goals.PUT("/:id/key-results/:kr_id", handleUpdateKeyResult(cfg.Queries))
			goals.DELETE("/:id/key-results/:kr_id", handleDeleteKeyResult(cfg.Queries))
			goals.GET("/:id/snapshots", handleListSnapshots(cfg.Queries))
		}

		// Performance Reviews
		reviews := protected.Group("/reviews")
		reviews.Use(RequireRole("boss"))
		{
			reviews.GET("/cycles", handleListReviewCycles(cfg.Queries))
			reviews.POST("/cycles", handleCreateReviewCycle(cfg.Queries))
			reviews.PUT("/cycles/:id", handleUpdateReviewCycle(cfg.Queries))
			reviews.DELETE("/cycles/:id", handleDeleteReviewCycle(cfg.Queries))
			reviews.GET("/cycles/:id/reviews", handleListReviewsByCycle(cfg.Queries))
			reviews.POST("/cycles/:id/reviews", handleCreateReview(cfg.Queries))
			reviews.PUT("/cycles/:id/reviews/:review_id", handleUpdateReview(cfg.Queries))
		}

		// 1:1 Meetings
		meetings := protected.Group("/meetings")
		meetings.Use(RequireRole("boss"))
		{
			meetings.GET("", handleListMeetings(cfg.Queries))
			meetings.GET("/:id", handleGetMeeting(cfg.Queries))
			meetings.POST("", handleCreateMeeting(cfg.Queries))
			meetings.PUT("/:id", handleUpdateMeeting(cfg.Queries))
			meetings.DELETE("/:id", handleDeleteMeeting(cfg.Queries))
			meetings.GET("/:id/actions", handleListActionItems(cfg.Queries))
			meetings.POST("/:id/actions", handleCreateActionItem(cfg.Queries))
			meetings.PUT("/:id/actions/:ai_id", handleUpdateActionItem(cfg.Queries))
			meetings.DELETE("/:id/actions/:ai_id", handleDeleteActionItem(cfg.Queries))
			meetings.GET("/actions/open", handleListOpenActionItems(cfg.Queries))
		}

		// Skills
		skills := protected.Group("/skills")
		skills.Use(RequireRole("boss"))
		{
			skills.GET("", handleListSkills(cfg.Queries))
			skills.POST("", handleCreateSkill(cfg.Queries))
			skills.PUT("/:id", handleUpdateSkill(cfg.Queries))
			skills.DELETE("/:id", handleDeleteSkill(cfg.Queries))
			skills.GET("/matrix", handleGetSkillMatrix(cfg.Queries))
			skills.GET("/employees/:emp_id", handleListEmployeeSkills(cfg.Queries))
			skills.POST("/employees/:emp_id", handleSetEmployeeSkill(cfg.Queries))
			skills.DELETE("/employees/:emp_id/:skill_id", handleDeleteEmployeeSkill(cfg.Queries))
		}

		// Training Programs
		training := protected.Group("/training")
		training.Use(RequireRole("boss"))
		{
			training.GET("", handleListTrainingPrograms(cfg.Queries))
			training.POST("", handleCreateTrainingProgram(cfg.Queries))
			training.PUT("/:id", handleUpdateTrainingProgram(cfg.Queries))
			training.DELETE("/:id", handleDeleteTrainingProgram(cfg.Queries))
			training.GET("/:id/enrollments", handleListEnrollments(cfg.Queries))
			training.POST("/:id/enrollments", handleCreateEnrollment(cfg.Queries))
			training.PUT("/:id/enrollments/:eid", handleUpdateEnrollment(cfg.Queries))
			training.DELETE("/:id/enrollments/:eid", handleDeleteEnrollment(cfg.Queries))
		}

		// Career Paths
		career := protected.Group("/career")
		career.Use(RequireRole("boss"))
		{
			career.GET("/levels", handleListCareerLevels(cfg.Queries))
			career.POST("/levels", handleCreateCareerLevel(cfg.Queries))
			career.PUT("/levels/:id", handleUpdateCareerLevel(cfg.Queries))
			career.DELETE("/levels/:id", handleDeleteCareerLevel(cfg.Queries))
			career.GET("/paths", handleListCareerPaths(cfg.Queries))
			career.POST("/paths", handleUpsertCareerPath(cfg.Queries))
			career.DELETE("/paths/:id", handleDeleteCareerPath(cfg.Queries))
		}

		// Metrics / KPIs
		kpis := protected.Group("/kpis")
		kpis.Use(RequireRole("boss"))
		{
			kpis.GET("", handleListMetrics(cfg.Queries))
			kpis.POST("", handleCreateMetric(cfg.Queries))
			kpis.GET("/dashboard", handleGetMetricsWithValues(cfg.Queries))
			kpis.PUT("/:id", handleUpdateMetric(cfg.Queries))
			kpis.DELETE("/:id", handleDeleteMetric(cfg.Queries))
			kpis.POST("/:id/values", handleIngestMetricValue(cfg.Queries))
			kpis.GET("/:id/values", handleListMetricValues(cfg.Queries))
		}

		// Projects
		projects := protected.Group("/projects")
		projects.Use(RequireRole("boss"))
		{
			projects.GET("", handleListProjects(cfg.Queries))
			projects.POST("", handleCreateProject(cfg.Queries))
			projects.GET("/:id", handleGetProject(cfg.Queries))
			projects.PUT("/:id", handleUpdateProject(cfg.Queries))
			projects.DELETE("/:id", handleDeleteProject(cfg.Queries))
		}

		// Tasks
		tasks := protected.Group("/tasks")
		tasks.Use(RequireRole("boss"))
		{
			tasks.GET("", handleListTasks(cfg.Queries))
			tasks.POST("", handleCreateTask(cfg.Queries))
			tasks.GET("/overdue", handleListOverdueTasks(cfg.Queries))
			tasks.GET("/stats", handleCountTasksByStatus(cfg.Queries))
			tasks.GET("/:id", handleGetTask(cfg.Queries))
			tasks.PUT("/:id", handleUpdateTask(cfg.Queries))
			tasks.DELETE("/:id", handleDeleteTask(cfg.Queries))
		}

		// Recommendations (AI recommendation engine)
		if cfg.Dispatcher != nil {
			recs := protected.Group("/recommendations")
			recs.Use(RequireRole("boss"))
			{
				recs.GET("", handleListRecommendations(cfg.Queries))
				recs.GET("/summary", handleGetRecommendationSummary(cfg.Queries))
				recs.POST("/:id/execute", handleExecuteRecommendation(cfg.Queries, cfg.Dispatcher))
				recs.POST("/:id/execute-all", handleExecuteAllRecommendation(cfg.Queries, cfg.Dispatcher))
				recs.POST("/:id/dismiss", handleDismissRecommendation(cfg.Queries))
				recs.DELETE("/:id", handleDeleteRecommendation(cfg.Queries))
			}
		}

		// Reporting Lines
		reporting := protected.Group("/reporting-lines")
		reporting.Use(RequireRole("boss"))
		{
			reporting.GET("", handleListReportingLines(cfg.Queries))
			reporting.POST("", handleCreateReportingLine(cfg.Queries))
			reporting.DELETE("/:id", handleDeleteReportingLine(cfg.Queries))
			reporting.GET("/reports/:manager_id", handleGetDirectReports(cfg.Queries))
		}

		// Workflows
		workflows := protected.Group("/workflows")
		workflows.Use(RequireRole("boss"))
		{
			workflows.GET("", handleListWorkflows(cfg.Queries))
			workflows.POST("", handleCreateWorkflow(cfg.Queries))
			workflows.PUT("/:id", handleUpdateWorkflow(cfg.Queries))
			workflows.DELETE("/:id", handleDeleteWorkflow(cfg.Queries))
		}

		// Incentives
		incentives := protected.Group("/incentives")
		incentives.Use(RequireRole("boss"))
		{
			incentives.GET("/rules", handleListIncentiveRules(cfg.Queries))
			incentives.POST("/rules", handleCreateIncentiveRule(cfg.Queries))
			incentives.PUT("/rules/:id", handleUpdateIncentiveRule(cfg.Queries))
			incentives.DELETE("/rules/:id", handleDeleteIncentiveRule(cfg.Queries))
			incentives.GET("/scores", handleListIncentiveScores(cfg.Queries))
		}

		// State & Signals
		state := protected.Group("/state")
		state.Use(RequireRole("boss"))
		{
			state.GET("", handleGetCompanyState(cfg.Queries))
			state.GET("/events", handleListCommunicationEvents(cfg.Queries))
			state.GET("/signals", handleListExecutionSignals(cfg.Queries))
			state.GET("/risks", handleGetTopRisks(cfg.Queries))
			state.GET("/memory", handleGetWorkingMemory(cfg.Queries))
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

		// Write operations (send messages via bot/channels)
		if cfg.ActionService != nil {
			actionHandler := NewOpenClawActionHandler(cfg.ActionService)
			openclaw.POST("/checkin", actionHandler.HandleCheckin)
			openclaw.POST("/chase", actionHandler.HandleChase)
			openclaw.POST("/summary", actionHandler.HandleSummary)
			openclaw.POST("/message", actionHandler.HandleMessage)
		}
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

		// Brain Layer v2 (MCP accessible — under /openclaw/ to avoid route conflicts)
		mcpAPI.GET("/openclaw/state", handleGetCompanyState(cfg.Queries))
		mcpAPI.GET("/openclaw/state/events", handleListCommunicationEvents(cfg.Queries))
		mcpAPI.GET("/openclaw/state/signals", handleListExecutionSignals(cfg.Queries))
		mcpAPI.GET("/openclaw/state/risks", handleGetTopRisks(cfg.Queries))
		mcpAPI.GET("/openclaw/state/memory", handleGetWorkingMemory(cfg.Queries))
		mcpAPI.GET("/openclaw/kpis", handleGetMetricsWithValues(cfg.Queries))
		mcpAPI.GET("/openclaw/tasks/overdue", handleListOverdueTasks(cfg.Queries))
		mcpAPI.GET("/openclaw/tasks/stats", handleCountTasksByStatus(cfg.Queries))
		mcpAPI.GET("/openclaw/incentives/scores", handleListIncentiveScores(cfg.Queries))
	}

	// Webhook endpoints (signature-verified, no JWT)
	webhookVerifier := NewWebhookVerifier()
	billingHandler := NewBillingHandler(cfg.Billing, webhookVerifier)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/stripe", webhookVerifier.VerifyMiddleware("stripe"), billingHandler.HandleStripeWebhook)

		halaosHandler := NewHalaOSWebhookHandler(cfg.Queries, cfg.HalaOSMapper)
		webhooks.POST("/halaos", halaosHandler.HandleWebhook)
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
