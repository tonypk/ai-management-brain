package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// parseUUID converts a string UUID to pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %w", err)
	}
	return u, nil
}

// parseDate parses a YYYY-MM-DD string into pgtype.Date.
func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// generateInviteCode creates a short random uppercase hex string.
func generateInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// --- Tenant handlers ---

// handleGetTenant returns the authenticated user's tenant info.
func handleGetTenant(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		tenant, err := queries.GetTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			slog.Error("get tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Parse mentor_blend if present
		var mentorBlend *brain.BlendConfig
		if len(tenant.MentorBlend) > 0 {
			var bc brain.BlendConfig
			if err := json.Unmarshal(tenant.MentorBlend, &bc); err == nil && bc.PrimaryID != "" {
				mentorBlend = &bc
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":           formatUUID(tenant.ID),
				"name":         tenant.Name,
				"timezone":     tenant.Timezone,
				"mentor_id":    tenant.MentorID,
				"mentor_blend": mentorBlend,
			},
		})
	}
}

// updateTenantRequest holds the request body for updating a tenant.
type updateTenantRequest struct {
	Name     string `json:"name" binding:"required,min=1"`
	Timezone string `json:"timezone" binding:"required,min=1"`
}

// handleUpdateTenant updates the tenant's name and timezone.
func handleUpdateTenant(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateTenantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and timezone are required"})
			return
		}

		// Validate timezone
		if _, err := time.LoadLocation(req.Timezone); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid timezone: %s", req.Timezone)})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		if err := queries.UpdateTenantNameTimezone(c.Request.Context(), sqlc.UpdateTenantNameTimezoneParams{
			ID:       tenantID,
			Name:     req.Name,
			Timezone: req.Timezone,
		}); err != nil {
			slog.Error("update tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"name": req.Name, "timezone": req.Timezone}})
	}
}

// --- Employee handlers ---

// handleListEmployees lists active employees for the tenant.
func handleListEmployees(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		employees, err := queries.ListActiveEmployees(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		result := make([]gin.H, 0, len(employees))
		for _, e := range employees {
			result = append(result, gin.H{
				"id":           formatUUID(e.ID),
				"name":         e.Name,
				"culture_code": e.CultureCode,
				"role":         e.Role,
				"is_active":    e.IsActive,
				"has_telegram":  e.TelegramID.Valid,
				"invite_code":  e.InviteCode.String,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// createEmployeeRequest holds the request body for creating an employee.
type createEmployeeRequest struct {
	Name        string `json:"name" binding:"required,min=1"`
	CultureCode string `json:"culture_code"`
}

// handleCreateEmployee creates a new employee with an invite code.
func handleCreateEmployee(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createEmployeeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		cultureCode := req.CultureCode
		if cultureCode == "" {
			cultureCode = "default"
		}

		// Validate culture code
		if !brain.ValidCultures[cultureCode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid culture code: %s", cultureCode)})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		inviteCode := generateInviteCode()

		emp, err := queries.CreateEmployee(c.Request.Context(), sqlc.CreateEmployeeParams{
			TenantID:    tenantID,
			Name:        req.Name,
			CultureCode: cultureCode,
			Role:        "member",
			InviteCode:  pgtype.Text{String: inviteCode, Valid: true},
		})
		if err != nil {
			slog.Error("create employee", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{
				"id":           formatUUID(emp.ID),
				"name":         emp.Name,
				"culture_code": emp.CultureCode,
				"invite_code":  inviteCode,
			},
		})
	}
}

// handleGetEmployee returns a single employee by ID.
func handleGetEmployee(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		empID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee ID"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		emp, err := queries.GetEmployee(c.Request.Context(), sqlc.GetEmployeeParams{
			ID:       empID,
			TenantID: tenantID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
				return
			}
			slog.Error("get employee", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":           formatUUID(emp.ID),
				"name":         emp.Name,
				"culture_code": emp.CultureCode,
				"role":         emp.Role,
				"is_active":    emp.IsActive,
				"has_telegram":  emp.TelegramID.Valid,
				"invite_code":  emp.InviteCode.String,
			},
		})
	}
}

// updateCultureRequest holds the request body for updating an employee's culture.
type updateCultureRequest struct {
	CultureCode string `json:"culture_code" binding:"required,min=1"`
}

// handleUpdateEmployeeCulture updates an employee's culture code.
func handleUpdateEmployeeCulture(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateCultureRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "culture_code is required"})
			return
		}

		if !brain.ValidCultures[req.CultureCode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid culture code: %s", req.CultureCode)})
			return
		}

		empID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee ID"})
			return
		}

		// Verify employee belongs to tenant
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		if _, err := queries.GetEmployee(c.Request.Context(), sqlc.GetEmployeeParams{
			ID:       empID,
			TenantID: tenantID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
				return
			}
			slog.Error("get employee for culture update", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if err := queries.UpdateEmployeeCulture(c.Request.Context(), sqlc.UpdateEmployeeCultureParams{
			ID:          empID,
			CultureCode: req.CultureCode,
		}); err != nil {
			slog.Error("update employee culture", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"culture_code": req.CultureCode}})
	}
}

// --- Report handlers ---

// handleListReports lists reports for a date (query param ?date=2024-01-01).
func handleListReports(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		dateStr := c.Query("date")
		if dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter is required (YYYY-MM-DD)"})
			return
		}

		reportDate, err := parseDate(dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		reports, err := queries.GetReportsByTenantDate(c.Request.Context(), sqlc.GetReportsByTenantDateParams{
			TenantID:   tenantID,
			ReportDate: reportDate,
		})
		if err != nil {
			slog.Error("list reports", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

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
			}
			if r.Blockers.Valid {
				item["blockers"] = r.Blockers.String
			}
			if r.Sentiment.Valid {
				item["sentiment"] = r.Sentiment.String
			}
			result = append(result, item)
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

// --- Summary handler ---

// handleGetSummary gets the summary for a date (query param ?date=2024-01-01).
func handleGetSummary(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		dateStr := c.Query("date")
		if dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date query parameter is required (YYYY-MM-DD)"})
			return
		}

		summaryDate, err := parseDate(dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		summary, err := queries.GetSummary(c.Request.Context(), sqlc.GetSummaryParams{
			TenantID:    tenantID,
			SummaryDate: summaryDate,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "summary not found for this date"})
				return
			}
			slog.Error("get summary", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		var keyMetrics interface{}
		if len(summary.KeyMetrics) > 0 {
			_ = json.Unmarshal(summary.KeyMetrics, &keyMetrics)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":              formatUUID(summary.ID),
				"summary_date":    summary.SummaryDate.Time.Format("2006-01-02"),
				"content":         summary.Content,
				"submission_rate": summary.SubmissionRate,
				"blockers_count":  summary.BlockersCount,
				"key_metrics":     keyMetrics,
			},
		})
	}
}

// --- Mentor handlers ---

// mentorInfo holds display information for a mentor.
type mentorInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

var mentorDescriptions = map[string]mentorInfo{
	"inamori": {ID: "inamori", Name: "Kazuo Inamori", Description: "Kyocera - Amoeba management, respect heaven and love people"},
	"dalio":   {ID: "dalio", Name: "Ray Dalio", Description: "Bridgewater - Radical transparency, principles-driven"},
	"grove":   {ID: "grove", Name: "Andy Grove", Description: "Intel - OKR-driven, high output management"},
	"ren":     {ID: "ren", Name: "Ren Zhengfei", Description: "Huawei - Wolf culture, self-criticism, striver-oriented"},
	"son":     {ID: "son", Name: "Masayoshi Son", Description: "SoftBank - 300-year vision, time machine theory"},
	"jobs":    {ID: "jobs", Name: "Steve Jobs", Description: "Apple - Pursuit of simplicity, reality distortion field"},
	"bezos":   {ID: "bezos", Name: "Jeff Bezos", Description: "Amazon - Day 1 mentality, customer obsession"},
	"ma":      {ID: "ma", Name: "Jack Ma", Description: "Alibaba - Embrace change, customer first, teamwork"},
}

// handleGetMentor returns the current mentor and available mentors list.
func handleGetMentor(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		tenant, err := queries.GetTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			slog.Error("get tenant for mentor", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		var currentBlend *brain.BlendConfig
		if len(tenant.MentorBlend) > 0 {
			var bc brain.BlendConfig
			if err := json.Unmarshal(tenant.MentorBlend, &bc); err == nil && bc.PrimaryID != "" {
				currentBlend = &bc
			}
		}

		available := make([]mentorInfo, 0, len(mentorDescriptions))
		for _, m := range mentorDescriptions {
			available = append(available, m)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"current_mentor_id": tenant.MentorID,
				"current_blend":     currentBlend,
				"available_mentors": available,
			},
		})
	}
}

// updateMentorRequest holds the request body for switching mentor.
type updateMentorRequest struct {
	MentorID string `json:"mentor_id" binding:"required"`
}

// handleUpdateMentor switches the active mentor.
func handleUpdateMentor(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateMentorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mentor_id is required"})
			return
		}

		if !brain.ValidMentors[req.MentorID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid mentor_id: %s", req.MentorID)})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		if err := queries.UpdateTenantMentor(c.Request.Context(), sqlc.UpdateTenantMentorParams{
			ID:       tenantID,
			MentorID: req.MentorID,
		}); err != nil {
			slog.Error("update mentor", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": gin.H{"mentor_id": req.MentorID}})
	}
}

// updateBlendRequest holds the request body for setting blend config.
type updateBlendRequest struct {
	PrimaryID   string `json:"primary_id" binding:"required"`
	SecondaryID string `json:"secondary_id" binding:"required"`
	Weight      int    `json:"weight" binding:"required,min=50,max=90"`
}

// handleUpdateBlend sets the mentor blend configuration.
func handleUpdateBlend(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateBlendRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "primary_id, secondary_id, and weight (50-90) are required"})
			return
		}

		if !brain.ValidMentors[req.PrimaryID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid primary_id: %s", req.PrimaryID)})
			return
		}
		if !brain.ValidMentors[req.SecondaryID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid secondary_id: %s", req.SecondaryID)})
			return
		}
		if req.PrimaryID == req.SecondaryID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "primary and secondary mentors must be different"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		blend := brain.BlendConfig{
			PrimaryID:   req.PrimaryID,
			SecondaryID: req.SecondaryID,
			Weight:      float64(req.Weight) / 100.0,
		}
		blendJSON, err := json.Marshal(blend)
		if err != nil {
			slog.Error("marshal blend", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if err := queries.UpdateTenantMentor(c.Request.Context(), sqlc.UpdateTenantMentorParams{
			ID:          tenantID,
			MentorID:    req.PrimaryID,
			MentorBlend: blendJSON,
		}); err != nil {
			slog.Error("update blend", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"primary_id":   req.PrimaryID,
				"secondary_id": req.SecondaryID,
				"weight":       req.Weight,
			},
		})
	}
}

// --- Dashboard handler ---

// handleDashboardStats returns dashboard statistics.
func handleDashboardStats(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Get tenant info for mentor
		tenant, err := queries.GetTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
				return
			}
			slog.Error("dashboard: get tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Count active employees
		employees, err := queries.ListActiveEmployees(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("dashboard: list employees", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Count today's submissions
		today := pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true}
		submissionCount, err := queries.CountReportsByTenantDate(c.Request.Context(), sqlc.CountReportsByTenantDateParams{
			TenantID:   tenantID,
			ReportDate: today,
		})
		if err != nil {
			slog.Error("dashboard: count reports", "error", err)
			submissionCount = 0
		}

		// Get latest summary date
		var lastSummaryDate string
		latestSummary, err := queries.GetLatestSummary(c.Request.Context(), tenantID)
		if err == nil {
			lastSummaryDate = latestSummary.SummaryDate.Time.Format("2006-01-02")
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"employee_count":    len(employees),
				"today_submissions": submissionCount,
				"current_mentor":    tenant.MentorID,
				"last_summary_date": lastSummaryDate,
			},
		})
	}
}
