package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- State & Signals ---

func handleListCommunicationEvents(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		since := time.Now().AddDate(0, 0, -7)
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		limit := int32(50)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = int32(n)
			}
		}

		events, err := q.ListCommunicationEvents(c.Request.Context(), sqlc.ListCommunicationEventsParams{
			TenantID:   tenantID,
			OccurredAt: pgtype.Timestamptz{Time: since, Valid: true},
			Limit:      limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list events"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": events})
	}
}

func handleListExecutionSignals(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		since := time.Now().AddDate(0, 0, -7)
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		limit := int32(50)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = int32(n)
			}
		}

		signals, err := q.ListExecutionSignals(c.Request.Context(), sqlc.ListExecutionSignalsParams{
			TenantID:    tenantID,
			GeneratedAt: pgtype.Timestamptz{Time: since, Valid: true},
			Limit:       limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list signals"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": signals})
	}
}

func handleGetTopRisks(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		since := time.Now().AddDate(0, 0, -7)
		limit := int32(10)

		risks, err := q.GetTopRisks(c.Request.Context(), sqlc.GetTopRisksParams{
			TenantID:    tenantID,
			GeneratedAt: pgtype.Timestamptz{Time: since, Valid: true},
			Limit:       limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get risks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": risks})
	}
}

func handleGetWorkingMemory(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		snapshotType := c.DefaultQuery("type", "daily_summary")
		snapshot, err := q.GetLatestSnapshot(c.Request.Context(), sqlc.GetLatestSnapshotParams{
			TenantID:     tenantID,
			SnapshotType: snapshotType,
		})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": snapshot})
	}
}

func handleGetCompanyState(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		ctx := c.Request.Context()
		since := time.Now().AddDate(0, 0, -7)
		sinceTz := pgtype.Timestamptz{Time: since, Valid: true}

		// Gather state in parallel-ish (sequential for simplicity)
		risks, _ := q.GetTopRisks(ctx, sqlc.GetTopRisksParams{
			TenantID: tenantID, GeneratedAt: sinceTz, Limit: 5,
		})
		overdue, _ := q.ListOverdueTasks(ctx, tenantID)
		taskStats, _ := q.CountTasksByStatus(ctx, tenantID)
		eventCounts, _ := q.CountEventsByType(ctx, sqlc.CountEventsByTypeParams{
			TenantID: tenantID, OccurredAt: sinceTz,
		})
		blocked, _ := q.ListBlockedProjects(ctx, tenantID)
		latestMemory, _ := q.GetLatestSnapshot(ctx, sqlc.GetLatestSnapshotParams{
			TenantID: tenantID, SnapshotType: "daily_summary",
		})

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"top_risks":        risks,
			"overdue_tasks":    overdue,
			"task_stats":       taskStats,
			"event_counts":     eventCounts,
			"blocked_projects": blocked,
			"working_memory":   latestMemory,
		}})
	}
}
