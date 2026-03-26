package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Recommendation handlers ---

// handleListRecommendations lists recommendations filtered by status/category with pagination.
func handleListRecommendations(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		status := c.DefaultQuery("status", "")
		category := c.DefaultQuery("category", "")

		limit := int32(20)
		if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
		offset := int32(0)
		if o, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && o >= 0 {
			offset = int32(o)
		}

		recs, err := q.ListRecommendations(c.Request.Context(), sqlc.ListRecommendationsParams{
			TenantID: tenantID,
			Column2:  status,
			Column3:  category,
			Limit:    limit,
			Offset:   offset,
		})
		if err != nil {
			slog.Error("list recommendations", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recommendations"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": recs})
	}
}

// handleGetRecommendationSummary returns top 3 pending recommendations + total pending count.
func handleGetRecommendationSummary(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		top3, err := q.GetRecommendationSummary(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get recommendation summary", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recommendation summary"})
			return
		}

		count, err := q.CountPendingRecommendations(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("count pending recommendations", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count pending recommendations"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"top":           top3,
				"pending_count": count,
			},
		})
	}
}

// executeRecommendationRequest holds the request body for executing a single action.
type executeRecommendationRequest struct {
	ActionIndex int `json:"action_index"`
}

// handleExecuteRecommendation executes a single action by index and marks the recommendation as "accepted".
func handleExecuteRecommendation(q *sqlc.Queries, dispatcher *brain.Dispatcher, feedback *brain.RecommendationFeedback) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		recID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req executeRecommendationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action_index is required"})
			return
		}

		rec, err := q.GetRecommendation(c.Request.Context(), sqlc.GetRecommendationParams{
			ID:       recID,
			TenantID: tenantID,
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "recommendation not found"})
			return
		}

		var actions []brain.SuggestedAction
		if err := json.Unmarshal(rec.SuggestedActions, &actions); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse actions"})
			return
		}

		if req.ActionIndex < 0 || req.ActionIndex >= len(actions) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action_index out of range"})
			return
		}

		result := dispatcher.Execute(c.Request.Context(), tenantID, actions[req.ActionIndex])

		// Mark as accepted on single action execution
		if err := q.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
			ID:       recID,
			TenantID: tenantID,
			Status:   "accepted",
		}); err != nil {
			slog.Error("update recommendation status", "error", err)
		}

		// Record feedback as strategy_result memory
		if feedback != nil {
			feedback.RecordFeedback(c.Request.Context(), TenantFromContext(c), rec.Title, actions[req.ActionIndex].Type, "accepted")
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// handleExecuteAllRecommendation runs all auto-executable actions and marks as "executed" if all succeed.
func handleExecuteAllRecommendation(q *sqlc.Queries, dispatcher *brain.Dispatcher, feedback *brain.RecommendationFeedback) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		recID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		rec, err := q.GetRecommendation(c.Request.Context(), sqlc.GetRecommendationParams{
			ID:       recID,
			TenantID: tenantID,
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "recommendation not found"})
			return
		}

		results := dispatcher.ExecuteAll(c.Request.Context(), tenantID, rec.SuggestedActions)

		// Check if all auto-executable actions succeeded
		allSuccess := true
		for _, r := range results {
			if r.Error != "" {
				allSuccess = false
				break
			}
		}

		status := "accepted"
		if allSuccess {
			status = "executed"
		}

		if err := q.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
			ID:       recID,
			TenantID: tenantID,
			Status:   status,
		}); err != nil {
			slog.Error("update recommendation status", "error", err)
		}

		// Record feedback as strategy_result memory
		if feedback != nil {
			feedback.RecordFeedback(c.Request.Context(), TenantFromContext(c), rec.Title, "execute_all", status)
		}

		c.JSON(http.StatusOK, gin.H{"data": results})
	}
}

// handleDismissRecommendation updates a recommendation status to "dismissed".
func handleDismissRecommendation(q *sqlc.Queries, feedback *brain.RecommendationFeedback) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		recID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		// Fetch title for feedback before status change
		var recTitle string
		if feedback != nil {
			rec, err := q.GetRecommendation(c.Request.Context(), sqlc.GetRecommendationParams{
				ID: recID, TenantID: tenantID,
			})
			if err == nil {
				recTitle = rec.Title
			}
		}

		if err := q.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
			ID:       recID,
			TenantID: tenantID,
			Status:   "dismissed",
		}); err != nil {
			slog.Error("dismiss recommendation", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dismiss recommendation"})
			return
		}

		if feedback != nil && recTitle != "" {
			feedback.RecordFeedback(c.Request.Context(), TenantFromContext(c), recTitle, "dismiss", "dismissed")
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "dismissed"}})
	}
}

// handleDeleteRecommendation deletes a recommendation (only dismissed/expired allowed).
func handleDeleteRecommendation(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		recID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		if err := q.DeleteRecommendation(c.Request.Context(), sqlc.DeleteRecommendationParams{
			ID:       recID,
			TenantID: tenantID,
		}); err != nil {
			slog.Error("delete recommendation", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recommendation"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
