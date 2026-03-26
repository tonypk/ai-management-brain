package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Projects ---

type createProjectRequest struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	OwnerID         string   `json:"owner_id"`
	OwnerTeamID     string   `json:"owner_team_id"`
	Status          string   `json:"status"`
	Priority        string   `json:"priority"`
	LinkedGoalIDs   []string `json:"linked_goal_ids"`
	LinkedMetricIDs []string `json:"linked_metric_ids"`
	SourceSystem    string   `json:"source_system"`
	SourceRef       string   `json:"source_ref"`
	StartDate       string   `json:"start_date"`
	DueDate         string   `json:"due_date"`
}

type updateProjectRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	OwnerID     string   `json:"owner_id"`
	Blockers    []string `json:"blockers"`
	DueDate     string   `json:"due_date"`
}

func handleListProjects(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		projects, err := q.ListProjects(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list projects"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": projects})
	}
}

func handleGetProject(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyProjectTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		project, err := q.GetProject(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get project"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": project})
	}
}

func handleCreateProject(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}
		var ownerTeamID pgtype.UUID
		if req.OwnerTeamID != "" {
			ownerTeamID, _ = parseUUID(req.OwnerTeamID)
		}

		status := req.Status
		if status == "" {
			status = "planned"
		}
		priority := req.Priority
		if priority == "" {
			priority = "medium"
		}

		goalIDsJSON, _ := jsonMarshal(req.LinkedGoalIDs)
		metricIDsJSON, _ := jsonMarshal(req.LinkedMetricIDs)

		var startDate pgtype.Date
		if req.StartDate != "" {
			startDate, _ = parseDate(req.StartDate)
		}
		var dueDate pgtype.Date
		if req.DueDate != "" {
			dueDate, _ = parseDate(req.DueDate)
		}

		project, err := q.CreateProject(c.Request.Context(), sqlc.CreateProjectParams{
			TenantID:        tenantID,
			Name:            req.Name,
			Description:     pgtype.Text{String: req.Description, Valid: req.Description != ""},
			OwnerID:         ownerID,
			OwnerTeamID:     ownerTeamID,
			Status:          status,
			Priority:        priority,
			LinkedGoalIds:   goalIDsJSON,
			LinkedMetricIds: metricIDsJSON,
			SourceSystem:    pgtype.Text{String: req.SourceSystem, Valid: req.SourceSystem != ""},
			SourceRef:       pgtype.Text{String: req.SourceRef, Valid: req.SourceRef != ""},
			StartDate:       startDate,
			DueDate:         dueDate,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": project})
	}
}

func handleUpdateProject(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyProjectTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req updateProjectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}

		blockersJSON, _ := jsonMarshal(req.Blockers)

		var dueDate pgtype.Date
		if req.DueDate != "" {
			dueDate, _ = parseDate(req.DueDate)
		}

		project, err := q.UpdateProject(c.Request.Context(), sqlc.UpdateProjectParams{
			ID:          id,
			Name:        req.Name,
			Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
			Status:      req.Status,
			Priority:    req.Priority,
			OwnerID:     ownerID,
			Blockers:    blockersJSON,
			DueDate:     dueDate,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update project"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": project})
	}
}

func handleDeleteProject(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyProjectTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteProject(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete project"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
