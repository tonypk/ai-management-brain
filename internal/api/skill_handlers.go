package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Request types ---

type createSkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type updateSkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type setEmployeeSkillRequest struct {
	SkillID string `json:"skill_id" binding:"required"`
	Level   int16  `json:"level" binding:"required,min=1,max=5"`
	Notes   string `json:"notes"`
}

// --- Handlers ---

func handleListSkills(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		skills, err := q.ListSkills(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list skills", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list skills"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": skills})
	}
}

func handleCreateSkill(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cat := req.Category
		if cat == "" {
			cat = "general"
		}

		skill, err := q.CreateSkill(c.Request.Context(), sqlc.CreateSkillParams{
			TenantID:    tenantID,
			Name:        req.Name,
			Category:    cat,
			Description: req.Description,
		})
		if err != nil {
			slog.Error("create skill", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create skill"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": skill})
	}
}

func handleUpdateSkill(q *sqlc.Queries) gin.HandlerFunc {
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
		var req updateSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cat := req.Category
		if cat == "" {
			cat = "general"
		}

		if err := q.UpdateSkill(c.Request.Context(), sqlc.UpdateSkillParams{
			ID:          id,
			TenantID:    tenantID,
			Name:        req.Name,
			Category:    cat,
			Description: req.Description,
		}); err != nil {
			slog.Error("update skill", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update skill"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleDeleteSkill(q *sqlc.Queries) gin.HandlerFunc {
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
		if err := q.DeleteSkill(c.Request.Context(), sqlc.DeleteSkillParams{ID: id, TenantID: tenantID}); err != nil {
			slog.Error("delete skill", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete skill"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleListEmployeeSkills(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("emp_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
			return
		}
		skills, err := q.ListEmployeeSkills(c.Request.Context(), empID)
		if err != nil {
			slog.Error("list employee skills", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list employee skills"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": skills})
	}
}

func handleSetEmployeeSkill(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("emp_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
			return
		}
		var req setEmployeeSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		skillID, err := parseUUID(req.SkillID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill_id"})
			return
		}

		es, err := q.SetEmployeeSkill(c.Request.Context(), sqlc.SetEmployeeSkillParams{
			EmployeeID: empID,
			SkillID:    skillID,
			Level:      req.Level,
			Notes:      req.Notes,
		})
		if err != nil {
			slog.Error("set employee skill", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set employee skill"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": es})
	}
}

func handleDeleteEmployeeSkill(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("emp_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
			return
		}
		skillID, err := parseUUID(c.Param("skill_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill_id"})
			return
		}
		if err := q.DeleteEmployeeSkill(c.Request.Context(), sqlc.DeleteEmployeeSkillParams{
			EmployeeID: empID,
			SkillID:    skillID,
		}); err != nil {
			slog.Error("delete employee skill", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete employee skill"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	}
}

func handleGetSkillMatrix(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		matrix, err := q.GetSkillMatrix(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get skill matrix", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get skill matrix"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": matrix})
	}
}
