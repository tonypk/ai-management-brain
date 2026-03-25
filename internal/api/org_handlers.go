package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
	"github.com/tonypk/ai-management-brain/internal/roles"
)

// --- Request types ---

type startWizardRequest struct {
	MentorID string `json:"mentor_id" binding:"required"`
}

type wizardAnswerRequest struct {
	Answer string `json:"answer" binding:"required"`
}

type updatePlanRequest struct {
	Feedback string `json:"feedback" binding:"required"`
}

type setupOrgRequest struct {
	Industry        string   `json:"industry" binding:"required"`
	CompanyStage    string   `json:"company_stage" binding:"required"`
	BusinessModel   string   `json:"business_model"`
	TeamSize        int      `json:"team_size" binding:"required,min=1,max=100000"`
	OrgStructure    string   `json:"org_structure" binding:"required"`
	CurrentProjects string   `json:"current_projects"`
	PainPoints      []string `json:"pain_points" binding:"required,min=1"`
	CommTools       []string `json:"comm_tools" binding:"required,min=1"`
	CulturePrefs    string   `json:"culture_prefs"`
	GoalFramework   string   `json:"goal_framework"`
}

// --- Handlers ---

// handleStartWizard is deprecated. Use the onboarding flow via Telegram/Slack/Lark instead.
// Kept as a stub to preserve API routes until frontend migration.
func handleStartWizard(queries *sqlc.Queries, wizard *brain.OrgWizard) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusGone, gin.H{"error": "wizard API is deprecated, use the onboarding flow via bot channels"})
	}
}

// handleWizardAnswer is deprecated. Use the onboarding flow via Telegram/Slack/Lark instead.
// Kept as a stub to preserve API routes until frontend migration.
func handleWizardAnswer(queries *sqlc.Queries, wizard *brain.OrgWizard) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusGone, gin.H{"error": "wizard API is deprecated, use the onboarding flow via bot channels"})
	}
}

// handleGetPlan returns the current management plan for the tenant.
func handleGetPlan(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		org, err := queries.GetOrganizationByTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no organization plan found"})
				return
			}
			slog.Error("get organization", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		var plan brain.ManagementPlan
		if err := json.Unmarshal(org.ManagementPlan, &plan); err != nil {
			slog.Error("unmarshal management plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":           formatUUID(org.ID),
				"industry":     org.Industry,
				"size":         org.Size,
				"stage":        org.Stage,
				"mentor_id":    org.MentorID,
				"plan":         plan,
				"plan_version": org.PlanVersion,
				"status":       org.Status,
			},
		})
	}
}

// handleUpdatePlan adjusts the management plan based on user feedback.
func handleUpdatePlan(queries *sqlc.Queries, engine *brain.OrgEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if engine == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI features not available"})
			return
		}

		var req updatePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "feedback is required"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		org, err := queries.GetOrganizationByTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no organization plan found"})
				return
			}
			slog.Error("get organization", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		var currentPlan brain.ManagementPlan
		if err := json.Unmarshal(org.ManagementPlan, &currentPlan); err != nil {
			slog.Error("unmarshal current plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		mentor, err := brain.LoadMentor(org.MentorID)
		if err != nil {
			slog.Error("load mentor", "mentor_id", org.MentorID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mentor"})
			return
		}

		newPlan, err := engine.AdjustPlan(c.Request.Context(), mentor, &currentPlan, req.Feedback)
		if err != nil {
			slog.Error("adjust plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to adjust plan"})
			return
		}

		planJSON, err := json.Marshal(newPlan)
		if err != nil {
			slog.Error("marshal new plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if err := queries.UpdateOrganizationPlan(c.Request.Context(), sqlc.UpdateOrganizationPlanParams{
			TenantID:       tenantID,
			ManagementPlan: planJSON,
		}); err != nil {
			slog.Error("update organization plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"plan":         newPlan,
				"plan_version": org.PlanVersion + 1,
			},
		})
	}
}

// matchIndustryFromHistory scans wizard conversation history and the current answer
// for industry keywords and returns the matching template (or nil).
func matchIndustryFromHistory(history []brain.WizardMessage, currentAnswer string) *brain.IndustryTemplate {
	// Check current answer first (most recent context)
	if tmpl := brain.MatchIndustry(currentAnswer); tmpl != nil {
		return tmpl
	}
	// Scan previous user messages
	for _, msg := range history {
		if msg.Role == "user" {
			if tmpl := brain.MatchIndustry(msg.Content); tmpl != nil {
				return tmpl
			}
		}
	}
	return nil
}

// handleActivatePlan changes the plan status from draft to active
// and triggers AI role creation from the plan's support roles.
func handleActivatePlan(queries *sqlc.Queries, roleManager *roles.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		org, err := queries.GetOrganizationByTenant(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no organization plan found"})
				return
			}
			slog.Error("get organization", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if org.Status == "active" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan is already active"})
			return
		}

		if err := queries.UpdateOrganizationStatus(c.Request.Context(), sqlc.UpdateOrganizationStatusParams{
			TenantID: tenantID,
			Status:   "active",
		}); err != nil {
			slog.Error("activate plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Activate AI roles from plan (non-fatal)
		rolesActivated := 0
		if roleManager != nil {
			var plan brain.ManagementPlan
			if err := json.Unmarshal(org.ManagementPlan, &plan); err != nil {
				slog.Error("unmarshal plan for roles", "error", err)
			} else {
				tenantIDStr := formatUUID(org.TenantID)
				if err := roleManager.ActivateForTenant(c.Request.Context(), tenantIDStr, &plan, org.MentorID); err != nil {
					slog.Error("activate AI roles", "error", err)
				} else {
					rolesActivated = len(roleManager.ListAgents())
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":          "active",
				"roles_activated": rolesActivated,
			},
		})
	}
}

// handleSetupOrg creates a new organization plan from structured form data.
// Uses OrgEngine.GeneratePlan() to produce a ManagementPlan via AI.
func handleSetupOrg(queries *sqlc.Queries, engine *brain.OrgEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if engine == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI features not available"})
			return
		}

		var req setupOrgRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Load tenant to get mentor_id
		tenant, err := queries.GetTenant(c.Request.Context(), tenantID)
		if err != nil {
			slog.Error("get tenant", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		mentor, err := brain.LoadMentor(tenant.MentorID)
		if err != nil {
			slog.Error("load mentor", "mentor_id", tenant.MentorID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mentor"})
			return
		}

		profile := brain.CompanyProfile{
			Industry:        req.Industry,
			Size:            req.TeamSize,
			Stage:           req.CompanyStage,
			BusinessModel:   req.BusinessModel,
			PainPoints:      req.PainPoints,
			OrgStructure:    req.OrgStructure,
			CurrentProjects: req.CurrentProjects,
			CommTools:       req.CommTools,
			CulturePrefs:    req.CulturePrefs,
			GoalFramework:   req.GoalFramework,
		}

		// Optional: match industry template for richer context
		industry := brain.MatchIndustry(req.Industry)

		plan, err := engine.GeneratePlan(c.Request.Context(), mentor, profile, industry)
		if err != nil {
			slog.Error("generate plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate plan"})
			return
		}

		planJSON, err := json.Marshal(plan)
		if err != nil {
			slog.Error("marshal plan", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Persist form metadata as JSON for JSONB columns
		currentProjectsJSON, _ := json.Marshal(req.CurrentProjects)
		teamStructureJSON, _ := json.Marshal(req.OrgStructure)
		culturePrefsJSON, _ := json.Marshal(req.CulturePrefs)

		org, err := queries.UpsertOrganization(c.Request.Context(), sqlc.UpsertOrganizationParams{
			TenantID:             tenantID,
			Industry:             pgtype.Text{String: req.Industry, Valid: true},
			Size:                 pgtype.Int4{Int32: int32(req.TeamSize), Valid: true},
			Stage:                pgtype.Text{String: req.CompanyStage, Valid: true},
			BusinessModel:        pgtype.Text{String: req.BusinessModel, Valid: req.BusinessModel != ""},
			MentorID:             tenant.MentorID,
			ManagementPlan:       planJSON,
			ManagementPainPoints: req.PainPoints,
			CurrentProjects:      currentProjectsJSON,
			TargetFramework:      pgtype.Text{String: req.GoalFramework, Valid: req.GoalFramework != ""},
			TeamStructure:        teamStructureJSON,
			CommunicationTools:   req.CommTools,
			CulturePreferences:   culturePrefsJSON,
		})
		if err != nil {
			slog.Error("upsert organization", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"id":           formatUUID(org.ID),
				"industry":     org.Industry,
				"size":         org.Size,
				"stage":        org.Stage,
				"mentor_id":    org.MentorID,
				"plan":         plan,
				"plan_version": org.PlanVersion,
				"status":       org.Status,
			},
		})
	}
}
