package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Workflows ---

type createWorkflowRequest struct {
	Name             string      `json:"name" binding:"required"`
	Category         string      `json:"category"`
	TriggerConditions interface{} `json:"trigger_conditions"`
	Steps            interface{} `json:"steps"`
	ApprovalRules    interface{} `json:"approval_rules"`
	EscalationRules  interface{} `json:"escalation_rules"`
}

type updateWorkflowRequest struct {
	Name             string      `json:"name" binding:"required"`
	Category         string      `json:"category"`
	TriggerConditions interface{} `json:"trigger_conditions"`
	Steps            interface{} `json:"steps"`
	ApprovalRules    interface{} `json:"approval_rules"`
	EscalationRules  interface{} `json:"escalation_rules"`
}

func handleListWorkflows(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		workflows, err := q.ListWorkflows(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workflows"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": workflows})
	}
}

func handleCreateWorkflow(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createWorkflowRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		triggerJSON, _ := jsonMarshal(req.TriggerConditions)
		stepsJSON, _ := jsonMarshal(req.Steps)
		approvalJSON, _ := jsonMarshal(req.ApprovalRules)
		escalationJSON, _ := jsonMarshal(req.EscalationRules)

		wf, err := q.CreateWorkflow(c.Request.Context(), sqlc.CreateWorkflowParams{
			TenantID:          tenantID,
			Name:              req.Name,
			Category:          pgtype.Text{String: req.Category, Valid: req.Category != ""},
			TriggerConditions: triggerJSON,
			Steps:             stepsJSON,
			ApprovalRules:     approvalJSON,
			EscalationRules:   escalationJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workflow"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": wf})
	}
}

func handleUpdateWorkflow(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyWorkflowTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req updateWorkflowRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		triggerJSON, _ := jsonMarshal(req.TriggerConditions)
		stepsJSON, _ := jsonMarshal(req.Steps)
		approvalJSON, _ := jsonMarshal(req.ApprovalRules)
		escalationJSON, _ := jsonMarshal(req.EscalationRules)

		wf, err := q.UpdateWorkflow(c.Request.Context(), sqlc.UpdateWorkflowParams{
			ID:                id,
			Name:              req.Name,
			Category:          pgtype.Text{String: req.Category, Valid: req.Category != ""},
			TriggerConditions: triggerJSON,
			Steps:             stepsJSON,
			ApprovalRules:     approvalJSON,
			EscalationRules:   escalationJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workflow"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": wf})
	}
}

func handleDeleteWorkflow(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyWorkflowTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteWorkflow(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workflow"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
