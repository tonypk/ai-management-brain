package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/memory"
	"github.com/tonypk/ai-management-brain/internal/scheduler"
)

// --- Channel handlers ---

// handleGetChannels returns the tenant's channel configuration and router status.
func handleGetChannels(queries *sqlc.Queries, router *channel.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		cfg, err := queries.GetTenantChannelConfig(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get channel config", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Build per-channel status
		channels := []gin.H{
			{
				"type":       "telegram",
				"configured": true,
			},
			{
				"type":       "signal",
				"configured": cfg.SignalPhone.Valid && cfg.SignalPhone.String != "",
			},
			{
				"type":       "slack",
				"configured": cfg.SlackBotToken.Valid && cfg.SlackBotToken.String != "",
			},
			{
				"type":       "lark",
				"configured": cfg.LarkAppID.Valid && cfg.LarkAppID.String != "",
			},
		}

		// Get registered router types
		var registeredTypes []string
		if router != nil {
			for _, t := range router.Types() {
				registeredTypes = append(registeredTypes, string(t))
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"enabled_channels":    cfg.EnabledChannels,
				"channels":            channels,
				"registered_channels": registeredTypes,
			},
		})
	}
}

// updateChannelsRequest holds the request body for updating tenant channel config.
type updateChannelsRequest struct {
	SlackBotToken      *string  `json:"slack_bot_token"`
	SlackSigningSecret *string  `json:"slack_signing_secret"`
	LarkAppID          *string  `json:"lark_app_id"`
	LarkAppSecret      *string  `json:"lark_app_secret"`
	SignalPhone        *string  `json:"signal_phone"`
	EnabledChannels    []string `json:"enabled_channels"`
}

// handleUpdateChannels updates the tenant's channel configuration.
func handleUpdateChannels(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req updateChannelsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		// Build params with pgtype.Text wrappers
		params := sqlc.UpdateTenantChannelsParams{
			ID: tenantID,
		}
		if req.SlackBotToken != nil {
			params.SlackBotToken = pgtype.Text{String: *req.SlackBotToken, Valid: *req.SlackBotToken != ""}
		}
		if req.SlackSigningSecret != nil {
			params.SlackSigningSecret = pgtype.Text{String: *req.SlackSigningSecret, Valid: *req.SlackSigningSecret != ""}
		}
		if req.LarkAppID != nil {
			params.LarkAppID = pgtype.Text{String: *req.LarkAppID, Valid: *req.LarkAppID != ""}
		}
		if req.LarkAppSecret != nil {
			params.LarkAppSecret = pgtype.Text{String: *req.LarkAppSecret, Valid: *req.LarkAppSecret != ""}
		}
		if req.SignalPhone != nil {
			params.SignalPhone = pgtype.Text{String: *req.SignalPhone, Valid: *req.SignalPhone != ""}
		}
		if req.EnabledChannels != nil {
			params.EnabledChannels = req.EnabledChannels
		}

		if err := queries.UpdateTenantChannels(c.Request.Context(), params); err != nil {
			slog.Error("update channels", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": true}})
	}
}

// testChannelRequest holds the request body for testing a channel.
type testChannelRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Text   string `json:"text"`
}

// handleTestChannel sends a test message via a channel adapter.
func handleTestChannel(router *channel.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		if router == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "channel router not configured"})
			return
		}

		channelType := channel.Type(c.Param("channel"))
		var req testChannelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		text := req.Text
		if text == "" {
			text = "Test message from AI Management Brain admin panel"
		}

		if err := router.Send(c.Request.Context(), channelType, req.UserID, text); err != nil {
			slog.Error("test channel", "channel", channelType, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"sent": true, "channel": channelType}})
	}
}

// --- Employee channel handlers ---

// handleAdminListEmployees lists employees with their channel information.
func handleAdminListEmployees(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		employees, err := queries.ListEmployeesWithChannels(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("admin list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		result := make([]gin.H, 0, len(employees))
		for _, e := range employees {
			result = append(result, gin.H{
				"id":                formatUUID(e.ID),
				"name":              e.Name,
				"telegram_id":       e.TelegramID.Valid,
				"signal_phone":      e.SignalPhone.String,
				"slack_id":          e.SlackID.String,
				"lark_id":           e.LarkID.String,
				"preferred_channel": e.PreferredChannel,
				"culture_code":      e.CultureCode,
				"role":              e.Role,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// updateEmployeeChannelsRequest holds the request body for updating employee channels.
type updateEmployeeChannelsRequest struct {
	SignalPhone      *string `json:"signal_phone"`
	SlackID          *string `json:"slack_id"`
	LarkID           *string `json:"lark_id"`
	PreferredChannel *string `json:"preferred_channel"`
}

// handleUpdateEmployeeChannels updates an employee's channel identifiers.
func handleUpdateEmployeeChannels(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee ID"})
			return
		}

		var req updateEmployeeChannelsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		params := sqlc.UpdateEmployeeChannelsParams{
			ID: empID,
		}
		if req.SignalPhone != nil {
			params.SignalPhone = pgtype.Text{String: *req.SignalPhone, Valid: *req.SignalPhone != ""}
		}
		if req.SlackID != nil {
			params.SlackID = pgtype.Text{String: *req.SlackID, Valid: *req.SlackID != ""}
		}
		if req.LarkID != nil {
			params.LarkID = pgtype.Text{String: *req.LarkID, Valid: *req.LarkID != ""}
		}
		if req.PreferredChannel != nil {
			params.PreferredChannel = *req.PreferredChannel
		}

		if err := queries.UpdateEmployeeChannels(c.Request.Context(), params); err != nil {
			slog.Error("update employee channels", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": true}})
	}
}

// updateEmployeePreferredRequest holds the request body for updating preferred channel.
type updateEmployeePreferredRequest struct {
	PreferredChannel string `json:"preferred_channel" binding:"required"`
}

// validPreferredChannels contains the allowed preferred channel values.
var validPreferredChannels = map[string]bool{
	"telegram": true,
	"signal":   true,
	"slack":    true,
	"lark":     true,
}

// handleUpdateEmployeePreferred updates an employee's preferred channel.
func handleUpdateEmployeePreferred(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee ID"})
			return
		}

		var req updateEmployeePreferredRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "preferred_channel is required"})
			return
		}

		if !validPreferredChannels[req.PreferredChannel] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "preferred_channel must be telegram, signal, slack, or lark"})
			return
		}

		if err := queries.UpdateEmployeePreferredChannel(c.Request.Context(), sqlc.UpdateEmployeePreferredChannelParams{
			ID:               empID,
			PreferredChannel: req.PreferredChannel,
		}); err != nil {
			slog.Error("update employee preferred channel", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"preferred_channel": req.PreferredChannel}})
	}
}

// --- Report handlers ---

// handleAdminListReports lists reports with filtering and pagination.
func handleAdminListReports(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Parse pagination
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		// Parse filters
		var dateFrom, dateTo pgtype.Date
		if s := c.Query("date_from"); s != "" {
			d, err := parseDate(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			dateFrom = d
		}
		if s := c.Query("date_to"); s != "" {
			d, err := parseDate(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			dateTo = d
		}

		var employeeFilter pgtype.UUID
		if s := c.Query("employee_id"); s != "" {
			u, err := parseUUID(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee_id"})
				return
			}
			employeeFilter = u
		}

		channelFilter := c.Query("channel")

		// Query reports
		params := sqlc.ListReportsFilteredParams{
			TenantID: tenantID,
			Column2:  dateFrom,
			Column3:  dateTo,
			Column4:  employeeFilter,
			Column5:  channelFilter,
			Limit:    int32(limit),
			Offset:   int32(offset),
		}
		reports, err := queries.ListReportsFiltered(c.Request.Context(), params)
		if err != nil {
			slog.Error("admin list reports", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Count total
		total, err := queries.CountReportsFiltered(c.Request.Context(), sqlc.CountReportsFilteredParams{
			TenantID: tenantID,
			Column2:  dateFrom,
			Column3:  dateTo,
			Column4:  employeeFilter,
			Column5:  channelFilter,
		})
		if err != nil {
			slog.Error("count reports filtered", "error", err)
			total = 0
		}

		// Format results
		result := make([]gin.H, 0, len(reports))
		for _, r := range reports {
			var answers interface{}
			if len(r.Answers) > 0 {
				_ = json.Unmarshal(r.Answers, &answers)
			}

			item := gin.H{
				"id":            formatUUID(r.ID),
				"employee_id":   formatUUID(r.EmployeeID),
				"employee_name": r.EmployeeName,
				"report_date":   r.ReportDate.Time.Format("2006-01-02"),
				"answers":       answers,
				"submitted_at":  r.SubmittedAt.Time,
				"channel":       r.Channel,
			}
			if r.Blockers.Valid {
				item["blockers"] = r.Blockers.String
			}
			if r.Sentiment.Valid {
				item["sentiment"] = r.Sentiment.String
			}
			result = append(result, item)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": result,
			"meta": gin.H{
				"total":    total,
				"page":     page,
				"limit":    limit,
				"has_more": int64(offset+limit) < total,
			},
		})
	}
}

// handleReportStats returns report submission statistics by channel.
func handleReportStats(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var dateFrom, dateTo pgtype.Date
		if s := c.Query("date_from"); s != "" {
			d, err := parseDate(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			dateFrom = d
		}
		if s := c.Query("date_to"); s != "" {
			d, err := parseDate(s)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			dateTo = d
		}

		stats, err := queries.GetReportStatsByChannel(c.Request.Context(), sqlc.GetReportStatsByChannelParams{
			TenantID: tenantID,
			Column2:  dateFrom,
			Column3:  dateTo,
		})
		if err != nil {
			slog.Error("report stats", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}

// --- Mentor handler ---

// handleListMentors returns all available mentors.
func handleListMentors() gin.HandlerFunc {
	return func(c *gin.Context) {
		mentors := make([]mentorInfo, 0, len(mentorDescriptions))
		for _, m := range mentorDescriptions {
			mentors = append(mentors, m)
		}
		c.JSON(http.StatusOK, gin.H{"data": mentors})
	}
}

// --- Scheduler handlers ---

// handleListSchedulerJobs returns all scheduled jobs.
func handleListSchedulerJobs(sched *scheduler.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sched == nil {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": sched.ListJobs()})
	}
}

// updateJobScheduleRequest holds the request body for updating a job's cron schedule.
type updateJobScheduleRequest struct {
	Cron string `json:"cron" binding:"required"`
}

// handleUpdateJobSchedule updates a scheduled job's cron expression.
func handleUpdateJobSchedule(sched *scheduler.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sched == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "scheduler not configured"})
			return
		}

		jobName := c.Param("job")
		var req updateJobScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cron is required"})
			return
		}

		if err := sched.UpdateJobSchedule(jobName, req.Cron); err != nil {
			slog.Error("update job schedule", "job", jobName, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"job": jobName, "cron": req.Cron}})
	}
}

// handleTriggerJob manually triggers a scheduled job.
func handleTriggerJob(sched *scheduler.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if sched == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "scheduler not configured"})
			return
		}

		jobName := c.Param("job")
		if err := sched.TriggerJob(c.Request.Context(), jobName); err != nil {
			slog.Error("trigger job", "job", jobName, "error", err)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"triggered": jobName}})
	}
}

// --- Memory handler ---

// handleMemoryStats returns memory statistics for the tenant.
func handleMemoryStats(store *memory.MemoryStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := TenantFromContext(c)

		count, err := store.Count(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("memory stats", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"total": count}})
	}
}
