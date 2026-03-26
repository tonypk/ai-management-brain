package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// handleGetSyncManifest returns changed entities since last sync for a given storage type.
// GET /api/v1/openclaw/sync/manifest?storage_type=notion
func handleGetSyncManifest(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		storageType := c.Query("storage_type")
		if storageType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "storage_type is required"})
			return
		}

		// Determine "since" from last sync or default to 30 days ago
		since := time.Now().AddDate(0, 0, -30)
		cfg, err := q.GetSyncConfig(c.Request.Context(), sqlc.GetSyncConfigParams{
			TenantID:    tenantID,
			StorageType: storageType,
		})
		if err == nil && cfg.LastSyncAt.Valid {
			since = cfg.LastSyncAt.Time
		}

		sinceTz := pgtype.Timestamptz{Time: since, Valid: true}
		ctx := c.Request.Context()

		// Gather changed entities
		tasks, _ := q.GetChangedTasks(ctx, sqlc.GetChangedTasksParams{
			TenantID: tenantID, UpdatedAt: sinceTz,
		})
		goals, _ := q.GetChangedGoals(ctx, sqlc.GetChangedGoalsParams{
			TenantID: tenantID, UpdatedAt: sinceTz,
		})
		projects, _ := q.GetChangedProjects(ctx, sqlc.GetChangedProjectsParams{
			TenantID: tenantID, UpdatedAt: sinceTz,
		})
		metrics, _ := q.GetChangedMetrics(ctx, sqlc.GetChangedMetricsParams{
			TenantID: tenantID, UpdatedAt: sinceTz,
		})

		// Gather export-only data (signals, recommendations, working memory)
		signalsSinceTz := pgtype.Timestamptz{Time: since, Valid: true}
		signals, _ := q.ListExecutionSignals(ctx, sqlc.ListExecutionSignalsParams{
			TenantID: tenantID, GeneratedAt: signalsSinceTz, Limit: 50,
		})
		recommendations, _ := q.ListRecommendations(ctx, sqlc.ListRecommendationsParams{
			TenantID: tenantID, Column2: "pending", Column3: "", Limit: 20, Offset: 0,
		})
		workingMemory, _ := q.GetLatestSnapshot(ctx, sqlc.GetLatestSnapshotParams{
			TenantID: tenantID, SnapshotType: "daily_summary",
		})

		// Ensure nil slices become empty arrays in JSON
		if tasks == nil {
			tasks = []sqlc.GetChangedTasksRow{}
		}
		if goals == nil {
			goals = []sqlc.GetChangedGoalsRow{}
		}
		if projects == nil {
			projects = []sqlc.GetChangedProjectsRow{}
		}
		if metrics == nil {
			metrics = []sqlc.GetChangedMetricsRow{}
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"since": since.Format(time.RFC3339),
			"changes": gin.H{
				"tasks":    tasks,
				"goals":    goals,
				"projects": projects,
				"metrics":  metrics,
			},
			"export_only": gin.H{
				"signals":         signals,
				"recommendations": recommendations,
				"working_memory":  workingMemory,
			},
		}})
	}
}

// handleReportSyncResult records the result of a sync operation.
// POST /api/v1/openclaw/sync/result
func handleReportSyncResult(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req struct {
			StorageType string `json:"storage_type" binding:"required"`
			ItemsPushed int32  `json:"items_pushed"`
			ItemsPulled int32  `json:"items_pulled"`
			Conflicts   int32  `json:"conflicts"`
			Errors      []string `json:"errors"`
			PulledItems []struct {
				EntityType string                 `json:"entity_type"`
				ExternalID string                 `json:"external_id"`
				Data       map[string]interface{} `json:"data"`
			} `json:"pulled_items"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()

		// Get sync config
		cfg, err := q.GetSyncConfig(ctx, sqlc.GetSyncConfigParams{
			TenantID:    tenantID,
			StorageType: req.StorageType,
		})
		if err != nil {
			slog.Error("get sync config for result", "error", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "sync config not found for storage type"})
			return
		}

		// Create sync log
		logEntry, err := q.CreateSyncLog(ctx, sqlc.CreateSyncLogParams{
			TenantID:     tenantID,
			SyncConfigID: cfg.ID,
			Direction:    "bidirectional",
		})
		if err != nil {
			slog.Error("create sync log", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sync log"})
			return
		}

		// Determine status
		status := "completed"
		if len(req.Errors) > 0 {
			status = "completed_with_errors"
		}

		// Marshal errors to JSON
		errorsJSON, _ := json.Marshal(req.Errors)
		if errorsJSON == nil {
			errorsJSON = []byte("[]")
		}

		summary := ""
		if req.ItemsPushed > 0 || req.ItemsPulled > 0 {
			summary = "pushed " + strconv.Itoa(int(req.ItemsPushed)) + ", pulled " + strconv.Itoa(int(req.ItemsPulled))
		}
		if req.Conflicts > 0 {
			summary += ", " + strconv.Itoa(int(req.Conflicts)) + " conflicts"
		}

		// Complete sync log
		if err := q.CompleteSyncLog(ctx, sqlc.CompleteSyncLogParams{
			ID:          logEntry.ID,
			Status:      status,
			ItemsPushed: req.ItemsPushed,
			ItemsPulled: req.ItemsPulled,
			Conflicts:   req.Conflicts,
			Errors:      errorsJSON,
			Summary:     pgtype.Text{String: summary, Valid: summary != ""},
		}); err != nil {
			slog.Error("complete sync log", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete sync log"})
			return
		}

		// Update last sync on config
		if err := q.UpdateLastSync(ctx, sqlc.UpdateLastSyncParams{
			ID:             cfg.ID,
			LastSyncAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
			LastSyncStatus: pgtype.Text{String: status, Valid: true},
		}); err != nil {
			slog.Error("update last sync", "error", err)
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"status":       status,
			"sync_log_id":  logEntry.ID,
			"items_pushed": req.ItemsPushed,
			"items_pulled": req.ItemsPulled,
			"conflicts":    req.Conflicts,
		}})
	}
}

// handleConfigureSync upserts a sync configuration for a given storage type.
// PUT /api/v1/openclaw/sync/config
func handleConfigureSync(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req struct {
			StorageType          string                 `json:"storage_type" binding:"required"`
			IsEnabled            bool                   `json:"is_enabled"`
			EntityTypes          []string               `json:"entity_types"`
			SyncFrequencyMinutes int32                  `json:"sync_frequency_minutes"`
			Config               map[string]interface{} `json:"config"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.SyncFrequencyMinutes <= 0 {
			req.SyncFrequencyMinutes = 30
		}
		if req.EntityTypes == nil {
			req.EntityTypes = []string{}
		}

		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config object"})
			return
		}
		if configJSON == nil {
			configJSON = []byte("{}")
		}

		cfg, err := q.CreateSyncConfig(c.Request.Context(), sqlc.CreateSyncConfigParams{
			TenantID:             tenantID,
			StorageType:          req.StorageType,
			IsEnabled:            req.IsEnabled,
			EntityTypes:          req.EntityTypes,
			SyncFrequencyMinutes: req.SyncFrequencyMinutes,
			Config:               configJSON,
		})
		if err != nil {
			slog.Error("upsert sync config", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save sync config"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": cfg})
	}
}

// handleListSyncConfigs lists all sync configurations for the tenant.
// GET /api/v1/openclaw/sync/configs
func handleListSyncConfigs(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		configs, err := q.ListSyncConfigs(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list sync configs", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sync configs"})
			return
		}
		if configs == nil {
			configs = []sqlc.SyncConfig{}
		}

		c.JSON(http.StatusOK, gin.H{"data": configs})
	}
}

// handleListSyncLogs lists sync logs for a given config.
// GET /api/v1/openclaw/sync/logs?config_id=uuid&limit=20
func handleListSyncLogs(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		configIDStr := c.Query("config_id")
		if configIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "config_id is required"})
			return
		}

		configID, err := parseUUID(configIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config_id"})
			return
		}

		limit := int32(20)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
				limit = int32(n)
			}
		}

		logs, err := q.ListSyncLogs(c.Request.Context(), sqlc.ListSyncLogsParams{
			SyncConfigID: configID,
			Limit:        limit,
		})
		if err != nil {
			slog.Error("list sync logs", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list sync logs"})
			return
		}
		if logs == nil {
			logs = []sqlc.SyncLog{}
		}

		c.JSON(http.StatusOK, gin.H{"data": logs})
	}
}
