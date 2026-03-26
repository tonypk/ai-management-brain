package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Tasks ---

type createTaskRequest struct {
	ProjectID      string `json:"project_id"`
	GoalID         string `json:"goal_id"`
	KeyResultID    string `json:"key_result_id"`
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description"`
	OwnerID        string `json:"owner_id"`
	OwnerTeamID    string `json:"owner_team_id"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	DueAt          string `json:"due_at"`
	SourceSystem   string `json:"source_system"`
	SourceRef      string `json:"source_ref"`
	CreatedByAgent bool   `json:"created_by_agent"`
}

type updateTaskRequest struct {
	Title       string `json:"title" binding:"required"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	OwnerID     string `json:"owner_id"`
	DueAt       string `json:"due_at"`
	Description string `json:"description"`
}

func handleListTasks(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		tasks, err := q.ListTasks(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": tasks})
	}
}

func handleGetTask(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyTaskTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		task, err := q.GetTask(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": task})
	}
}

func handleCreateTask(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var projectID, goalID, krID, ownerID, ownerTeamID pgtype.UUID
		if req.ProjectID != "" {
			projectID, _ = parseUUID(req.ProjectID)
		}
		if req.GoalID != "" {
			goalID, _ = parseUUID(req.GoalID)
		}
		if req.KeyResultID != "" {
			krID, _ = parseUUID(req.KeyResultID)
		}
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}
		if req.OwnerTeamID != "" {
			ownerTeamID, _ = parseUUID(req.OwnerTeamID)
		}

		status := req.Status
		if status == "" {
			status = "todo"
		}
		priority := req.Priority
		if priority == "" {
			priority = "medium"
		}

		var dueAt pgtype.Timestamptz
		if req.DueAt != "" {
			if t, err := time.Parse(time.RFC3339, req.DueAt); err == nil {
				dueAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
		}

		task, err := q.CreateTask(c.Request.Context(), sqlc.CreateTaskParams{
			TenantID:       tenantID,
			ProjectID:      projectID,
			GoalID:         goalID,
			KeyResultID:    krID,
			Title:          req.Title,
			Description:    pgtype.Text{String: req.Description, Valid: req.Description != ""},
			OwnerID:        ownerID,
			OwnerTeamID:    ownerTeamID,
			Status:         status,
			Priority:       priority,
			DueAt:          dueAt,
			SourceSystem:   pgtype.Text{String: req.SourceSystem, Valid: req.SourceSystem != ""},
			SourceRef:      pgtype.Text{String: req.SourceRef, Valid: req.SourceRef != ""},
			CreatedByAgent: req.CreatedByAgent,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": task})
	}
}

func handleUpdateTask(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyTaskTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req updateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}

		var dueAt pgtype.Timestamptz
		if req.DueAt != "" {
			if t, err := time.Parse(time.RFC3339, req.DueAt); err == nil {
				dueAt = pgtype.Timestamptz{Time: t, Valid: true}
			}
		}

		task, err := q.UpdateTask(c.Request.Context(), sqlc.UpdateTaskParams{
			ID:          id,
			Title:       req.Title,
			Status:      req.Status,
			Priority:    req.Priority,
			OwnerID:     ownerID,
			DueAt:       dueAt,
			Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": task})
	}
}

func handleDeleteTask(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyTaskTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteTask(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func handleListOverdueTasks(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		tasks, err := q.ListOverdueTasks(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list overdue tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": tasks})
	}
}

func handleCountTasksByStatus(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		stats, err := q.CountTasksByStatus(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}
