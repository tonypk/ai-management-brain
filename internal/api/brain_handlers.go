package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleGetCompanyContext returns the full company context from ContextService.
func handleGetCompanyContext(cs *brain.ContextService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cs == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "context service not available"})
			return
		}
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		ctx, err := cs.GetCompanyContext(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get company context", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get company context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": ctx})
	}
}

// handleCreateExecutionPlan generates an AI execution plan via ExecutionPlanner.
func handleCreateExecutionPlan(ep *brain.ExecutionPlanner) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ep == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "execution planner not available (ANTHROPIC_API_KEY required)"})
			return
		}
		var req struct {
			FocusArea string `json:"focus_area"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			// focus_area is optional — default to "overall" on bind error
			req.FocusArea = "overall"
		}
		if req.FocusArea == "" {
			req.FocusArea = "overall"
		}
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		plan, err := ep.Plan(c.Request.Context(), tenantID, req.FocusArea)
		if err != nil {
			slog.Error("create execution plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate execution plan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": plan})
	}
}

// handleCalculateIncentives triggers incentive preview calculation for a period.
func handleCalculateIncentives(ie *brain.IncentiveEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ie == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "incentive engine not available (ANTHROPIC_API_KEY required)"})
			return
		}
		var req struct {
			Period string `json:"period" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period is required (YYYY-MM format)"})
			return
		}
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		results, err := ie.Preview(c.Request.Context(), tenantID, req.Period)
		if err != nil {
			slog.Error("calculate incentives", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate incentives"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": results})
	}
}

// handleUpdateContext updates organization-level context fields:
// strategic_priorities, key_risks, and management_style_weights.
// Any field omitted from the request body is left unchanged (COALESCE semantics).
func handleUpdateContext(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			StrategicPriorities    []string           `json:"strategic_priorities"`
			KeyRisks               []string           `json:"key_risks"`
			ManagementStyleWeights map[string]float64 `json:"management_style_weights"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Marshal each optional field; nil JSON bytes → COALESCE keeps existing value.
		var spJSON, krJSON, mswJSON []byte
		if req.StrategicPriorities != nil {
			spJSON, err = json.Marshal(req.StrategicPriorities)
			if err != nil {
				slog.Error("marshal strategic_priorities", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		}
		if req.KeyRisks != nil {
			krJSON, err = json.Marshal(req.KeyRisks)
			if err != nil {
				slog.Error("marshal key_risks", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		}
		if req.ManagementStyleWeights != nil {
			mswJSON, err = json.Marshal(req.ManagementStyleWeights)
			if err != nil {
				slog.Error("marshal management_style_weights", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		}

		if err := q.UpdateOrganizationContext(c.Request.Context(), sqlc.UpdateOrganizationContextParams{
			TenantID:               tenantID,
			StrategicPriorities:    spJSON,
			KeyRisks:               krJSON,
			ManagementStyleWeights: mswJSON,
		}); err != nil {
			slog.Error("update organization context", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update context"})
			return
		}

		// Build response showing which fields were updated
		updated := make([]string, 0, 3)
		if spJSON != nil {
			updated = append(updated, "strategic_priorities")
		}
		if krJSON != nil {
			updated = append(updated, "key_risks")
		}
		if mswJSON != nil {
			updated = append(updated, "management_style_weights")
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "updated", "fields": updated}})
	}
}
