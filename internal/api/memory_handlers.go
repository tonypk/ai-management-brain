package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

// handleListMemories returns paginated memories for the tenant with optional filters.
func handleListMemories(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)
		memType := c.Query("type")
		memTier := c.Query("tier")
		employeeID := c.Query("employee_id")

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		memories, err := store.List(c.Request.Context(), tenantID, memType, memTier, employeeID, int32(limit), int32(offset))
		if err != nil {
			slog.Error("list memories", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list memories"})
			return
		}

		total, _ := store.Count(c.Request.Context(), tenantID)

		c.JSON(http.StatusOK, gin.H{
			"data": memories,
			"meta": gin.H{
				"total":    total,
				"page":     page,
				"limit":    limit,
				"has_more": int64(offset+limit) < total,
			},
		})
	}
}

// handleGetMemory returns a single memory by ID.
func handleGetMemory(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)
		id := c.Param("id")

		mem, err := store.Get(c.Request.Context(), id, tenantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": mem})
	}
}

// searchMemoriesRequest holds the request body for semantic memory search.
type searchMemoriesRequest struct {
	Query      string `json:"query" binding:"required"`
	EmployeeID string `json:"employee_id"`
	Limit      int    `json:"limit"`
}

// handleSearchMemories performs semantic search across memories.
func handleSearchMemories(memEngine *memory.MemoryEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)

		var req searchMemoriesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
			return
		}
		if req.Limit <= 0 || req.Limit > 20 {
			req.Limit = 10
		}

		result, err := memEngine.RecallForMentor(c.Request.Context(), tenantID, req.EmployeeID, req.Query)
		if err != nil {
			slog.Error("search memories", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// handleDeleteMemory deletes a memory by ID (boss only).
func handleDeleteMemory(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)
		id := c.Param("id")

		if err := store.Delete(c.Request.Context(), id, tenantID); err != nil {
			slog.Error("delete memory", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete memory"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
	}
}

// handleGetEmployeeProfile returns the AI-generated employee profile from memories.
func handleGetEmployeeProfile(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)
		employeeID := c.Param("id")

		profile, err := store.GetProfile(c.Request.Context(), tenantID, employeeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": profile})
	}
}

// triggerConsolidationRequest holds the request body for triggering consolidation.
type triggerConsolidationRequest struct {
	Task string `json:"task" binding:"required"`
}

// handleTriggerConsolidation triggers a memory consolidation task (boss only).
func handleTriggerConsolidation(memEngine *memory.MemoryEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req triggerConsolidationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task is required (clean, merge, rebuild)"})
			return
		}

		task := memory.ConsolidationTask(req.Task)
		if err := memEngine.RunConsolidation(c.Request.Context(), task); err != nil {
			slog.Error("trigger consolidation", "task", req.Task, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": req.Task}})
	}
}
