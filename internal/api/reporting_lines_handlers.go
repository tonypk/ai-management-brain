package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Reporting Lines ---

type createReportingLineRequest struct {
	ManagerID        string `json:"manager_id" binding:"required"`
	ReportID         string `json:"report_id" binding:"required"`
	RelationshipType string `json:"relationship_type"`
}

func handleListReportingLines(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		lines, err := q.ListReportingLines(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reporting lines"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": lines})
	}
}

func handleCreateReportingLine(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createReportingLineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		managerID, err := parseUUID(req.ManagerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manager_id"})
			return
		}
		reportID, err := parseUUID(req.ReportID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report_id"})
			return
		}
		relType := req.RelationshipType
		if relType == "" {
			relType = "direct"
		}
		line, err := q.CreateReportingLine(c.Request.Context(), sqlc.CreateReportingLineParams{
			TenantID:         tenantID,
			ManagerID:        managerID,
			ReportID:         reportID,
			RelationshipType: relType,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reporting line"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": line})
	}
}

func handleDeleteReportingLine(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		id, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		row, err := q.VerifyReportingLineTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteReportingLine(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func handleGetDirectReports(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		managerID, err := parseUUID(c.Param("manager_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid manager_id"})
			return
		}
		reports, err := q.GetDirectReports(c.Request.Context(), managerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get direct reports"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": reports})
	}
}
