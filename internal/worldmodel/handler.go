package worldmodel

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleGetOverview returns the world model overview for the tenant
func HandleGetOverview(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		overview, err := svc.GetOverview(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get overview"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": overview})
	}
}

// HandleGetSkills returns the team skills from the world model
func HandleGetSkills(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		skills, err := svc.GetTeamSkills(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get skills"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": skills})
	}
}

// HandleGetRelationships returns the team relationships from the world model
func HandleGetRelationships(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		rels, err := svc.GetTeamRelationships(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get relationships"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": rels})
	}
}

// HandleGetBlockers returns the team blockers from the world model
func HandleGetBlockers(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		blockers, err := svc.GetTeamBlockers(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get blockers"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": blockers})
	}
}

// HandleGetInsights returns the active insights from the world model
func HandleGetInsights(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		insights, err := svc.GetActiveInsights(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get insights"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": insights})
	}
}

// HandleGetEmployeeWorldModel returns the complete world model for a specific employee
func HandleGetEmployeeWorldModel(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		employeeID := c.Param("id")
		wm, err := svc.GetEmployeeWorldModel(c.Request.Context(), tenantID, employeeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get employee world model"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": wm})
	}
}
