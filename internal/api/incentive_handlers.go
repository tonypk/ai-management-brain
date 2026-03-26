package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Incentive Rules ---

type createIncentiveRuleRequest struct {
	Name             string      `json:"name" binding:"required"`
	RewardModel      string      `json:"reward_model"`
	PayoutCycle      string      `json:"payout_cycle"`
	AttributionRules interface{} `json:"attribution_rules"`
	PenaltyRules     interface{} `json:"penalty_rules"`
	ScoringFormula   interface{} `json:"scoring_formula"`
	AppliesTo        interface{} `json:"applies_to"`
}

type updateIncentiveRuleRequest struct {
	Name             string      `json:"name" binding:"required"`
	RewardModel      string      `json:"reward_model"`
	PayoutCycle      string      `json:"payout_cycle"`
	AttributionRules interface{} `json:"attribution_rules"`
	ScoringFormula   interface{} `json:"scoring_formula"`
	AppliesTo        interface{} `json:"applies_to"`
}

func handleListIncentiveRules(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		rules, err := q.ListIncentiveRules(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": rules})
	}
}

func handleCreateIncentiveRule(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createIncentiveRuleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rewardModel := req.RewardModel
		if rewardModel == "" {
			rewardModel = "individual"
		}
		payoutCycle := req.PayoutCycle
		if payoutCycle == "" {
			payoutCycle = "monthly"
		}

		attrJSON, _ := jsonMarshal(req.AttributionRules)
		penaltyJSON, _ := jsonMarshal(req.PenaltyRules)
		formulaJSON, _ := jsonMarshal(req.ScoringFormula)
		appliesToJSON, _ := jsonMarshal(req.AppliesTo)

		rule, err := q.CreateIncentiveRule(c.Request.Context(), sqlc.CreateIncentiveRuleParams{
			TenantID:         tenantID,
			Name:             req.Name,
			RewardModel:      rewardModel,
			PayoutCycle:      payoutCycle,
			AttributionRules: attrJSON,
			PenaltyRules:     penaltyJSON,
			ScoringFormula:   formulaJSON,
			AppliesTo:        appliesToJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": rule})
	}
}

func handleUpdateIncentiveRule(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyIncentiveRuleTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req updateIncentiveRuleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		attrJSON, _ := jsonMarshal(req.AttributionRules)
		formulaJSON, _ := jsonMarshal(req.ScoringFormula)
		appliesToJSON, _ := jsonMarshal(req.AppliesTo)

		rule, err := q.UpdateIncentiveRule(c.Request.Context(), sqlc.UpdateIncentiveRuleParams{
			ID:               id,
			Name:             req.Name,
			RewardModel:      req.RewardModel,
			PayoutCycle:      req.PayoutCycle,
			AttributionRules: attrJSON,
			ScoringFormula:   formulaJSON,
			AppliesTo:        appliesToJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": rule})
	}
}

func handleDeleteIncentiveRule(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyIncentiveRuleTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteIncentiveRule(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func handleListIncentiveScores(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		period := c.DefaultQuery("period", "")
		if period == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period required"})
			return
		}
		scores, err := q.ListIncentiveScores(c.Request.Context(), sqlc.ListIncentiveScoresParams{
			TenantID: tenantID,
			Period:   period,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scores"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": scores})
	}
}
