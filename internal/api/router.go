package api

import (
	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// NewRouter creates the API router with public and protected routes.
func NewRouter(queries *sqlc.Queries, jwtSecret []byte) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Public routes
	v1 := r.Group("/api/v1")
	authHandler := NewAuthHandler(queries, jwtSecret)
	v1.POST("/auth/login", authHandler.HandleLogin)
	v1.POST("/auth/register", authHandler.HandleRegister)

	// Protected routes
	protected := v1.Group("")
	protected.Use(AuthMiddleware(jwtSecret))
	{
		// Tenant
		protected.GET("/tenant", handleGetTenant(queries))
		protected.PUT("/tenant", RequireRole("boss"), handleUpdateTenant(queries))

		// Employees
		protected.GET("/employees", handleListEmployees(queries))
		protected.POST("/employees", RequireRole("boss"), handleCreateEmployee(queries))
		protected.GET("/employees/:id", handleGetEmployee(queries))
		protected.PUT("/employees/:id/culture", RequireRole("boss"), handleUpdateEmployeeCulture(queries))

		// Reports
		protected.GET("/reports", handleListReports(queries))
		protected.GET("/reports/summary", handleGetSummary(queries))

		// Mentor config
		protected.GET("/mentor", handleGetMentor(queries))
		protected.PUT("/mentor", RequireRole("boss"), handleUpdateMentor(queries))
		protected.PUT("/mentor/blend", RequireRole("boss"), handleUpdateBlend(queries))

		// Dashboard stats
		protected.GET("/dashboard", handleDashboardStats(queries))
	}

	return r
}
