package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleListAIRoles returns active AI roles with pending suggestion counts.
func handleListAIRoles(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		roleInstances, err := queries.ListActiveAIRoles(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list ai roles", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		pending, err := queries.ListPendingSuggestions(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list pending suggestions", "error", err)
			pending = nil
		}

		// Count pending per role
		pendingCounts := make(map[string]int)
		for _, s := range pending {
			pendingCounts[s.RoleID]++
		}

		type roleResponse struct {
			ID           string `json:"id"`
			RoleID       string `json:"role_id"`
			Title        string `json:"title"`
			MentorID     string `json:"mentor_id"`
			IsActive     bool   `json:"is_active"`
			PendingCount int    `json:"pending_count"`
			CreatedAt    string `json:"created_at"`
		}

		result := make([]roleResponse, 0, len(roleInstances))
		for _, r := range roleInstances {
			result = append(result, roleResponse{
				ID:           formatUUID(r.ID),
				RoleID:       r.RoleID,
				Title:        r.Title,
				MentorID:     r.MentorID,
				IsActive:     r.IsActive,
				PendingCount: pendingCounts[r.RoleID],
				CreatedAt:    r.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// handleListSuggestions returns pending suggestions for the tenant.
func handleListSuggestions(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		suggestions, err := queries.ListPendingSuggestions(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list suggestions", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		type suggestionResponse struct {
			ID         string  `json:"id"`
			RoleID     string  `json:"role_id"`
			RoleTitle  string  `json:"role_title"`
			Capability string  `json:"capability"`
			Title      string  `json:"title"`
			Content    string  `json:"content"`
			Status     string  `json:"status"`
			CreatedAt  string  `json:"created_at"`
			ReviewedAt *string `json:"reviewed_at"`
		}

		result := make([]suggestionResponse, 0, len(suggestions))
		for _, s := range suggestions {
			resp := suggestionResponse{
				ID:         formatUUID(s.ID),
				RoleID:     s.RoleID,
				RoleTitle:  s.RoleTitle,
				Capability: s.Capability,
				Title:      s.Title,
				Content:    s.Content,
				Status:     s.Status,
				CreatedAt:  s.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			}
			if s.ReviewedAt.Valid {
				t := s.ReviewedAt.Time.Format("2006-01-02T15:04:05Z")
				resp.ReviewedAt = &t
			}
			result = append(result, resp)
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// handleApproveSuggestion approves a pending suggestion.
func handleApproveSuggestion(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		suggestionID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid suggestion id"})
			return
		}

		if err := queries.UpdateSuggestionStatus(c.Request.Context(), sqlc.UpdateSuggestionStatusParams{
			Status:   "approved",
			ID:       suggestionID,
			TenantID: tenantID,
		}); err != nil {
			slog.Error("approve suggestion", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "approved"}})
	}
}

// handleRejectSuggestion rejects a pending suggestion.
func handleRejectSuggestion(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		suggestionID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid suggestion id"})
			return
		}

		if err := queries.UpdateSuggestionStatus(c.Request.Context(), sqlc.UpdateSuggestionStatusParams{
			Status:   "rejected",
			ID:       suggestionID,
			TenantID: tenantID,
		}); err != nil {
			slog.Error("reject suggestion", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "rejected"}})
	}
}

