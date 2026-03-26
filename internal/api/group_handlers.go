package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleListGroups returns all group chats for the tenant (including inactive).
func handleListGroups(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		groups, err := queries.ListGroupChatsByTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list groups", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		result := make([]gin.H, 0, len(groups))
		for _, g := range groups {
			result = append(result, gin.H{
				"id":               formatUUID(g.ID),
				"platform":         g.Platform,
				"platform_chat_id": g.PlatformChatID,
				"name":             g.Name,
				"group_type":       g.GroupType,
				"is_active":        g.IsActive,
				"created_at":       g.CreatedAt.Time,
				"updated_at":       g.UpdatedAt.Time,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// updateGroupRequest holds the request body for updating a group chat.
type updateGroupRequest struct {
	Name      string `json:"name" binding:"required,min=1"`
	GroupType string `json:"group_type" binding:"required,min=1"`
	IsActive  bool   `json:"is_active"`
}

// handleUpdateGroup updates a group chat's name, type, and active status.
func handleUpdateGroup(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateGroupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and group_type are required"})
			return
		}

		groupID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Verify group belongs to this tenant
		existing, err := queries.GetGroupChatByID(c.Request.Context(), groupID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
				return
			}
			slog.Error("get group for update", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if formatUUID(existing.TenantID) != formatUUID(tenantID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}

		updated, err := queries.UpdateGroupChat(c.Request.Context(), sqlc.UpdateGroupChatParams{
			ID:        groupID,
			Name:      req.Name,
			GroupType: req.GroupType,
			IsActive:  req.IsActive,
		})
		if err != nil {
			slog.Error("update group", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":               formatUUID(updated.ID),
				"platform":         updated.Platform,
				"platform_chat_id": updated.PlatformChatID,
				"name":             updated.Name,
				"group_type":       updated.GroupType,
				"is_active":        updated.IsActive,
				"updated_at":       updated.UpdatedAt.Time,
			},
		})
	}
}

// handleDeleteGroup soft-deletes a group chat.
func handleDeleteGroup(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group ID"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Verify group belongs to this tenant
		existing, err := queries.GetGroupChatByID(c.Request.Context(), groupID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
				return
			}
			slog.Error("get group for delete", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		if formatUUID(existing.TenantID) != formatUUID(tenantID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
			return
		}

		if err := queries.SoftDeleteGroupChat(c.Request.Context(), groupID); err != nil {
			slog.Error("delete group", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
	}
}
