package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Metrics ---

type createMetricRequest struct {
	Name             string   `json:"name" binding:"required"`
	DisplayName      string   `json:"display_name" binding:"required"`
	Formula          string   `json:"formula"`
	Unit             string   `json:"unit"`
	Source           string   `json:"source"`
	RefreshFrequency string   `json:"refresh_frequency"`
	TargetValue      *float64 `json:"target_value"`
	AlertThreshold   *float64 `json:"alert_threshold"`
	OwnerID          string   `json:"owner_id"`
	OwnerTeamID      string   `json:"owner_team_id"`
	Tags             []string `json:"tags"`
}

type updateMetricRequest struct {
	DisplayName    string   `json:"display_name" binding:"required"`
	Formula        string   `json:"formula"`
	Unit           string   `json:"unit"`
	Source         string   `json:"source"`
	TargetValue    *float64 `json:"target_value"`
	AlertThreshold *float64 `json:"alert_threshold"`
	OwnerID        string   `json:"owner_id"`
	Tags           []string `json:"tags"`
}

type ingestMetricValueRequest struct {
	ObservedAt string                 `json:"observed_at"`
	Value      float64                `json:"value" binding:"required"`
	Dimensions map[string]interface{} `json:"dimensions"`
	SourceRef  string                 `json:"source_ref"`
}

func handleListMetrics(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		metrics, err := q.ListMetrics(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list metrics"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": metrics})
	}
}

func handleGetMetricsWithValues(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		metrics, err := q.GetMetricsWithLatestValues(c.Request.Context(), tenantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics dashboard"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": metrics})
	}
}

func handleCreateMetric(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		var req createMetricRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}
		var ownerTeamID pgtype.UUID
		if req.OwnerTeamID != "" {
			ownerTeamID, _ = parseUUID(req.OwnerTeamID)
		}

		var targetVal pgtype.Numeric
		if req.TargetValue != nil {
			_ = targetVal.Scan(strconv.FormatFloat(*req.TargetValue, 'f', -1, 64))
		}
		var alertVal pgtype.Numeric
		if req.AlertThreshold != nil {
			_ = alertVal.Scan(strconv.FormatFloat(*req.AlertThreshold, 'f', -1, 64))
		}

		unit := req.Unit
		if unit == "" {
			unit = "%"
		}
		source := req.Source
		if source == "" {
			source = "manual"
		}

		tagsJSON, _ := jsonMarshal(req.Tags)

		metric, err := q.CreateMetric(c.Request.Context(), sqlc.CreateMetricParams{
			TenantID:         tenantID,
			Name:             req.Name,
			DisplayName:      req.DisplayName,
			Formula:          req.Formula,
			Unit:             pgtype.Text{String: unit, Valid: true},
			Source:           source,
			RefreshFrequency: pgtype.Text{String: req.RefreshFrequency, Valid: req.RefreshFrequency != ""},
			TargetValue:      targetVal,
			AlertThreshold:   alertVal,
			OwnerID:          ownerID,
			OwnerTeamID:      ownerTeamID,
			Tags:             tagsJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create metric"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": metric})
	}
}

func handleUpdateMetric(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyMetricTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req updateMetricRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != "" {
			ownerID, _ = parseUUID(req.OwnerID)
		}
		var targetVal pgtype.Numeric
		if req.TargetValue != nil {
			_ = targetVal.Scan(strconv.FormatFloat(*req.TargetValue, 'f', -1, 64))
		}
		var alertVal pgtype.Numeric
		if req.AlertThreshold != nil {
			_ = alertVal.Scan(strconv.FormatFloat(*req.AlertThreshold, 'f', -1, 64))
		}

		tagsJSON, _ := jsonMarshal(req.Tags)

		metric, err := q.UpdateMetric(c.Request.Context(), sqlc.UpdateMetricParams{
			ID:             id,
			DisplayName:    req.DisplayName,
			Formula:        req.Formula,
			Unit:           pgtype.Text{String: req.Unit, Valid: req.Unit != ""},
			Source:         req.Source,
			TargetValue:    targetVal,
			AlertThreshold: alertVal,
			OwnerID:        ownerID,
			Tags:           tagsJSON,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update metric"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": metric})
	}
}

func handleDeleteMetric(q *sqlc.Queries) gin.HandlerFunc {
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
		row, err := q.VerifyMetricTenant(c.Request.Context(), id)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := q.DeleteMetric(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete metric"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func handleIngestMetricValue(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		metricID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric id"})
			return
		}
		row, err := q.VerifyMetricTenant(c.Request.Context(), metricID)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var req ingestMetricValueRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		observedAt := time.Now()
		if req.ObservedAt != "" {
			if t, err := time.Parse(time.RFC3339, req.ObservedAt); err == nil {
				observedAt = t
			}
		}

		var valueNum pgtype.Numeric
		_ = valueNum.Scan(strconv.FormatFloat(req.Value, 'f', -1, 64))

		dimJSON, _ := jsonMarshal(req.Dimensions)

		mv, err := q.IngestMetricValue(c.Request.Context(), sqlc.IngestMetricValueParams{
			MetricID:   metricID,
			ObservedAt: pgtype.Timestamptz{Time: observedAt, Valid: true},
			Value:      valueNum,
			Dimensions: dimJSON,
			SourceRef:  pgtype.Text{String: req.SourceRef, Valid: req.SourceRef != ""},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest value"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": mv})
	}
}

func handleListMetricValues(q *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}
		metricID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric id"})
			return
		}
		row, err := q.VerifyMetricTenant(c.Request.Context(), metricID)
		if err != nil || row != tenantID {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		since := time.Now().AddDate(0, -1, 0) // default 30 days
		if s := c.Query("since"); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				since = t
			}
		}
		limit := int32(100)
		if l := c.Query("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 {
				limit = int32(n)
			}
		}

		values, err := q.ListMetricValues(c.Request.Context(), sqlc.ListMetricValuesParams{
			MetricID:   metricID,
			ObservedAt: pgtype.Timestamptz{Time: since, Valid: true},
			Limit:      limit,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list values"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": values})
	}
}
