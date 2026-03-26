package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Career Levels ---

type createCareerLevelRequest struct {
	Title        string `json:"title" binding:"required"`
	LevelOrder   int32  `json:"level_order"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
}

type updateCareerLevelRequest struct {
	Title        string `json:"title" binding:"required"`
	LevelOrder   int32  `json:"level_order"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
}

func handleListCareerLevels(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		levels, err := q.ListCareerLevels(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list levels"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": levels})
	}
}

func handleCreateCareerLevel(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createCareerLevelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		level, err := q.CreateCareerLevel(c.Request.Context(), sqlc.CreateCareerLevelParams{
			TenantID:     tenantID,
			Title:        req.Title,
			LevelOrder:   req.LevelOrder,
			Description:  req.Description,
			Requirements: req.Requirements,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create level"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": level})
	}
}

func handleUpdateCareerLevel(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		levelID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid level id"})
			return
		}
		owner, err := q.VerifyCareerLevelTenant(c.Request.Context(), levelID)
		if err != nil || owner != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "level not found"})
			return
		}
		var req updateCareerLevelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		level, err := q.UpdateCareerLevel(c.Request.Context(), sqlc.UpdateCareerLevelParams{
			ID:           levelID,
			Title:        req.Title,
			LevelOrder:   req.LevelOrder,
			Description:  req.Description,
			Requirements: req.Requirements,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update level"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": level})
	}
}

func handleDeleteCareerLevel(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		levelID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid level id"})
			return
		}
		owner, err := q.VerifyCareerLevelTenant(c.Request.Context(), levelID)
		if err != nil || owner != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "level not found"})
			return
		}
		if err := q.DeleteCareerLevel(c.Request.Context(), levelID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete level"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

// --- Career Paths ---

type upsertCareerPathRequest struct {
	EmployeeID     string `json:"employee_id" binding:"required"`
	CurrentLevelID string `json:"current_level_id"`
	TargetLevelID  string `json:"target_level_id"`
	TargetDate     string `json:"target_date"`
	Notes          string `json:"notes"`
}

func handleListCareerPaths(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		paths, err := q.ListCareerPaths(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list career paths"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": paths})
	}
}

func handleUpsertCareerPath(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req upsertCareerPathRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		empID, err := parseUUID(req.EmployeeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
			return
		}
		var currentLevelID, targetLevelID pgtype.UUID
		if req.CurrentLevelID != "" {
			currentLevelID, err = parseUUID(req.CurrentLevelID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current level id"})
				return
			}
		}
		if req.TargetLevelID != "" {
			targetLevelID, err = parseUUID(req.TargetLevelID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target level id"})
				return
			}
		}
		var targetDate pgtype.Date
		if req.TargetDate != "" {
			targetDate, err = parseDate(req.TargetDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target date"})
				return
			}
		}
		path, err := q.UpsertCareerPath(c.Request.Context(), sqlc.UpsertCareerPathParams{
			EmployeeID:     empID,
			CurrentLevelID: currentLevelID,
			TargetLevelID:  targetLevelID,
			TargetDate:     targetDate,
			Notes:          req.Notes,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert career path"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": path})
	}
}

func handleDeleteCareerPath(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
			return
		}
		if err := q.DeleteCareerPath(c.Request.Context(), pathID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete career path"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
