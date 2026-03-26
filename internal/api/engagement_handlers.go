package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleStartEngagement handles POST /consulting/start.
// Request body: {problem: string, mentor_id?: string, culture_code?: string}
func handleStartEngagement(ce *brain.ConsultingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available (ANTHROPIC_API_KEY required)"})
			return
		}
		var req struct {
			Problem     string `json:"problem" binding:"required"`
			MentorID    string `json:"mentor_id"`
			CultureCode string `json:"culture_code"`
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
		eng, firstQuestion, err := ce.StartEngagement(c.Request.Context(), tenantID, req.Problem, req.MentorID, req.CultureCode)
		if err != nil {
			slog.Error("start engagement", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start engagement"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": gin.H{
			"engagement_id":  eng.ID,
			"first_question": firstQuestion,
		}})
	}
}

// handleAnswerQuestion handles POST /consulting/:id/answer.
// Request body: {answer: string}
func handleAnswerQuestion(ce *brain.ConsultingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available"})
			return
		}
		var req struct {
			Answer string `json:"answer" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		nextQuestion, planText, done, err := ce.AnswerQuestion(c.Request.Context(), engagementID, req.Answer)
		if err != nil {
			slog.Error("answer question", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process answer"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"next_question": nextQuestion,
			"plan_text":     planText,
			"done":          done,
		}})
	}
}

// handleReviewAction handles POST /consulting/actions/:id/review.
// Request body: {approved: bool}
func handleReviewAction(ce *brain.ConsultingEngine, queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available"})
			return
		}
		var req struct {
			Approved bool `json:"approved"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		actionID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action id"})
			return
		}
		if err := ce.ReviewAction(c.Request.Context(), actionID, req.Approved); err != nil {
			slog.Error("review action", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to review action"})
			return
		}
		status := "rejected"
		if req.Approved {
			status = "approved"
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": status}})
	}
}

// handleExecuteApproved handles POST /consulting/:id/execute.
func handleExecuteApproved(ce *brain.ConsultingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available"})
			return
		}
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		results, err := ce.ExecuteApproved(c.Request.Context(), engagementID)
		if err != nil {
			slog.Error("execute approved", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to execute approved actions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"results": results}})
	}
}

// handleCheckProgress handles GET /consulting/:id/progress.
func handleCheckProgress(ce *brain.ConsultingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available"})
			return
		}
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		report, err := ce.CheckProgress(c.Request.Context(), engagementID)
		if err != nil {
			slog.Error("check progress", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check progress"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"report": report}})
	}
}

// handleCloseEngagement handles POST /consulting/:id/close.
func handleCloseEngagement(ce *brain.ConsultingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ce == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "consulting engine not available"})
			return
		}
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		summary, err := ce.CloseEngagement(c.Request.Context(), engagementID)
		if err != nil {
			slog.Error("close engagement", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close engagement"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"summary": summary}})
	}
}

// handleListEngagements handles GET /consulting.
// Lists active engagements for the current tenant.
func handleListEngagements(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		engagements, err := queries.ListActiveEngagements(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list engagements", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list engagements"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": engagements})
	}
}

// handleGetEngagement handles GET /consulting/:id.
// Returns a single engagement by ID.
func handleGetEngagement(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		engagement, err := queries.GetEngagement(c.Request.Context(), engagementID)
		if err != nil {
			slog.Error("get engagement", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get engagement"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": engagement})
	}
}

// handleListEngagementActions handles GET /consulting/:id/actions.
// Returns all actions for an engagement with linked task/meeting status.
func handleListEngagementActions(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		engagementID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engagement id"})
			return
		}
		actions, err := queries.ListEngagementActionsWithLinks(c.Request.Context(), engagementID)
		if err != nil {
			slog.Error("list engagement actions", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list engagement actions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": actions})
	}
}
