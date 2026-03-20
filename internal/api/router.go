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
	}

	return r
}
