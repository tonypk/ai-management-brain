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

// --- Handlers ---

// handleStartWizard begins a new wizard conversation.
func handleStartWizard(queries *sqlc.Queries, wizard *brain.OrgWizard) gin.HandlerFunc {
	return func(c *gin.Context) {
		if wizard == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI features not available"})
			return
		}

		var req startWizardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mentor_id is required"})
			return
		}

		if !brain.ValidMentors[req.MentorID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mentor_id"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		mentor, err := brain.LoadMentor(req.MentorID)
		if err != nil {
			slog.Error("load mentor", "mentor_id", req.MentorID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mentor"})
			return
		}

		// Start wizard conversation
		resp, err := wizard.Start(c.Request.Context(), mentor)
		if err != nil {
			slog.Error("wizard start", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start wizard"})
			return
		}

		// Save wizard session
		conversation := []brain.WizardMessage{
			{Role: "mentor", Content: resp.MentorMessage},
		}
		convJSON, _ := json.Marshal(conversation)
		profileJSON, _ := json.Marshal(map[string]interface{}{})

		// Delete any existing sessions for this tenant
		_ = queries.DeleteWizardSessions(c.Request.Context(), tenantID)

		session, err := queries.CreateWizardSession(c.Request.Context(), sqlc.CreateWizardSessionParams{
			TenantID:       tenantID,
			MentorID:       req.MentorID,
			CurrentStep:    "collecting",
			Conversation:   convJSON,
			CompanyProfile: profileJSON,
		})
		if err != nil {
			slog.Error("create wizard session", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"session_id": formatUUID(session.ID),
				"mentor_id":  req.MentorID,
				"message":    resp.MentorMessage,
				"is_complete": false,
			},
		})
	}
}

// handleWizardAnswer processes a user's answer in the wizard flow.
func handleWizardAnswer(queries *sqlc.Queries, wizard *brain.OrgWizard) gin.HandlerFunc {
	return func(c *gin.Context) {
		if wizard == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI features not available"})
			return
		}

		var req wizardAnswerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "answer is required"})
			return
		}

		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		// Get latest wizard session
		session, err := queries.GetLatestWizardSession(c.Request.Context(), tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no active wizard session, start one first"})
				return
			}
			slog.Error("get wizard session", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Load mentor
		mentor, err := brain.LoadMentor(session.MentorID)
		if err != nil {
			slog.Error("load mentor", "mentor_id", session.MentorID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mentor"})
			return
		}

		// Restore conversation history
		var history []brain.WizardMessage
		if err := json.Unmarshal(session.Conversation, &history); err != nil {
			slog.Error("unmarshal conversation", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Process the answer
		resp, err := wizard.ProcessAnswer(c.Request.Context(), mentor, history, req.Answer)
		if err != nil {
			slog.Error("wizard process answer", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process answer"})
			return
		}

		// Update conversation history
		history = append(history, brain.WizardMessage{Role: "user", Content: req.Answer})
		history = append(history, brain.WizardMessage{Role: "mentor", Content: resp.MentorMessage})
		convJSON, _ := json.Marshal(history)

		profileJSON := session.CompanyProfile
		step := "collecting"
		if resp.IsComplete && resp.Profile != nil {
			profileJSON, _ = json.Marshal(resp.Profile)
			step = "complete"
		}

		// Update session
		if err := queries.UpdateWizardSession(c.Request.Context(), sqlc.UpdateWizardSessionParams{
			ID:             session.ID,
			CurrentStep:    step,
			Conversation:   convJSON,
			CompanyProfile: profileJSON,
		}); err != nil {
			slog.Error("update wizard session", "error", err)
		}

		// If plan is generated, save organization
		if resp.IsComplete && resp.Plan != nil && resp.Profile != nil {
			planJSON, _ := json.Marshal(resp.Plan)

			// Delete existing org for this tenant (if any) then create new
			_ = queries.DeleteOrganization(c.Request.Context(), tenantID)

			_, err := queries.CreateOrganization(c.Request.Context(), sqlc.CreateOrganizationParams{
				TenantID:       tenantID,
				Industry:       resp.Profile.Industry,
				Size:           int32(resp.Profile.Size),
				Stage:          resp.Profile.Stage,
				BusinessModel:  pgtype.Text{String: resp.Profile.BusinessModel, Valid: resp.Profile.BusinessModel != ""},
				Region:         pgtype.Text{String: resp.Profile.Region, Valid: resp.Profile.Region != ""},
				MentorID:       session.MentorID,
				ManagementPlan: planJSON,
				Status:         "draft",
			})
			if err != nil {
				slog.Error("create organization", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save organization"})
				return
			}
		}

		result := gin.H{
			"message":     resp.MentorMessage,
			"is_complete": resp.IsComplete,
		}
		if resp.Plan != nil {
			result["plan"] = resp.Plan
		}
		if resp.Profile != nil {
			result["profile"] = resp.Profile
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
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
