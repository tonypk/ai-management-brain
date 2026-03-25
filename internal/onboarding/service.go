package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Querier is the subset of sqlc.Queries needed for onboarding.
type Querier interface {
	GetOnboardingSession(ctx context.Context, tenantID pgtype.UUID) (sqlc.OnboardingSession, error)
	CreateOnboardingSession(ctx context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error)
	UpdateOnboardingSession(ctx context.Context, arg sqlc.UpdateOnboardingSessionParams) error
	DeleteOnboardingSession(ctx context.Context, tenantID pgtype.UUID) error
	UpdateOrganizationFromOnboarding(ctx context.Context, arg sqlc.UpdateOrganizationFromOnboardingParams) error
	SetTenantOnboardingCompleted(ctx context.Context, id pgtype.UUID) error
	CreateOrgUnit(ctx context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error)
	DeleteOrgUnitsByTenant(ctx context.Context, tenantID pgtype.UUID) error
}

// RedisClient is the subset of go-redis/v9 commands needed for onboarding.
type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

const (
	lockTTL    = 60 * time.Second
	historyTTL = 24 * time.Hour

	statusOnboarding = "onboarding"
	statusConfiguring = "configuring"
	statusConfirming  = "confirming"
	statusActive      = "active"

	totalConfirmSteps = 4
)

// Service is the central state machine that orchestrates the onboarding flow.
type Service struct {
	db        Querier
	redis     RedisClient
	extractor *Extractor
	planner   *Planner
	confirmer *Confirmer
	chatLLM   brain.ChatLLMClient
}

// NewService creates a new onboarding Service.
func NewService(db Querier, rds RedisClient, extractLLM brain.LLMClient, planLLM brain.LLMClient, confirmLLM brain.LLMClient, chatLLM brain.ChatLLMClient) *Service {
	return &Service{
		db:        db,
		redis:     rds,
		extractor: NewExtractor(extractLLM),
		planner:   NewPlanner(planLLM),
		confirmer: NewConfirmer(confirmLLM),
		chatLLM:   chatLLM,
	}
}

// HandleMessage is the main entry point. It acquires a lock, loads the session,
// and routes to the appropriate handler based on the session status.
func (s *Service) HandleMessage(ctx context.Context, tenantID pgtype.UUID, channelType, userID, text string) (string, error) {
	// 1. Acquire processing lock via Redis SetNX.
	lockKey := fmt.Sprintf("onboarding:lock:%x", tenantID.Bytes)
	acquired, err := s.redis.SetNX(ctx, lockKey, "1", lockTTL).Result()
	if err != nil {
		slog.Warn("redis SetNX failed, proceeding without lock", "error", err)
	} else if !acquired {
		return "I'm still thinking about your last message, one moment...", nil
	}
	defer s.redis.Del(ctx, lockKey)

	// 2. Get or create session.
	session, err := s.getOrCreateSession(ctx, tenantID, channelType)
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}

	// 3. Route by state.
	switch session.Status {
	case statusOnboarding:
		return s.handleOnboarding(ctx, session, text)
	case statusConfiguring:
		return s.handleConfiguring(ctx, session)
	case statusConfirming:
		return s.handleConfirming(ctx, session, text)
	default:
		return "Onboarding is already complete.", nil
	}
}

// getOrCreateSession fetches an existing session or creates a new one.
func (s *Service) getOrCreateSession(ctx context.Context, tenantID pgtype.UUID, channelType string) (*sqlc.OnboardingSession, error) {
	session, err := s.db.GetOnboardingSession(ctx, tenantID)
	if err != nil {
		// Assume not found — create new session.
		session, err = s.db.CreateOnboardingSession(ctx, sqlc.CreateOnboardingSessionParams{
			TenantID:    tenantID,
			ChannelType: channelType,
		})
		if err != nil {
			return nil, err
		}
	}
	return &session, nil
}

// handleOnboarding processes messages during the onboarding (dialogue) state.
// It extracts info, checks completion, and generates a conversational response.
func (s *Service) handleOnboarding(ctx context.Context, session *sqlc.OnboardingSession, text string) (string, error) {
	// 1. Unmarshal collected data.
	collected := &CollectedData{}
	if len(session.CollectedData) > 0 {
		if err := json.Unmarshal(session.CollectedData, collected); err != nil {
			slog.Warn("failed to unmarshal collected_data", "error", err)
		}
	}

	// 2. Extract info from user message.
	updated, err := s.extractor.ExtractInfo(ctx, collected, text)
	if err != nil {
		return "", fmt.Errorf("extract info: %w", err)
	}

	newCount := session.MessageCount + 1

	// 3. Check if all required fields are covered -> transition to configuring.
	if updated.RequiredFieldsCovered() {
		collectedJSON, err := json.Marshal(updated)
		if err != nil {
			return "", fmt.Errorf("marshal collected data: %w", err)
		}

		if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
			TenantID:      session.TenantID,
			Status:        statusConfiguring,
			ConfirmStep:   0,
			CollectedData: collectedJSON,
			ProposedPlan:  session.ProposedPlan,
			MessageCount:  newCount,
			ChannelType:   session.ChannelType,
		}); err != nil {
			return "", fmt.Errorf("update session to configuring: %w", err)
		}

		// Immediately run configuring.
		configSession := &sqlc.OnboardingSession{
			ID:            session.ID,
			TenantID:      session.TenantID,
			Status:        statusConfiguring,
			ConfirmStep:   0,
			CollectedData: collectedJSON,
			ProposedPlan:  session.ProposedPlan,
			MessageCount:  newCount,
			ChannelType:   session.ChannelType,
		}
		return s.handleConfiguring(ctx, configSession)
	}

	// 4. Load chat history from Redis.
	history := s.loadChatHistory(ctx, session.TenantID)

	// 5. Generate conversational response with chat history.
	systemPrompt := BuildConsultantPrompt(updated, int(newCount))
	reply, err := s.chatLLM.ChatWithHistory(ctx, systemPrompt, history, text)
	if err != nil {
		return "", fmt.Errorf("chat LLM: %w", err)
	}

	// 6. Save updated history.
	history = append(history, brain.ChatMessage{Role: "user", Content: text})
	history = append(history, brain.ChatMessage{Role: "assistant", Content: reply})
	s.saveChatHistory(ctx, session.TenantID, history)

	// 7. Update session with new collected data and message count.
	collectedJSON, err := json.Marshal(updated)
	if err != nil {
		return "", fmt.Errorf("marshal collected data: %w", err)
	}

	if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
		TenantID:      session.TenantID,
		Status:        statusOnboarding,
		ConfirmStep:   session.ConfirmStep,
		CollectedData: collectedJSON,
		ProposedPlan:  session.ProposedPlan,
		MessageCount:  newCount,
		ChannelType:   session.ChannelType,
	}); err != nil {
		return "", fmt.Errorf("update session: %w", err)
	}

	return reply, nil
}

// handleConfiguring generates the management plan and transitions to confirming.
func (s *Service) handleConfiguring(ctx context.Context, session *sqlc.OnboardingSession) (string, error) {
	// 1. Unmarshal collected data.
	collected := &CollectedData{}
	if len(session.CollectedData) > 0 {
		if err := json.Unmarshal(session.CollectedData, collected); err != nil {
			return "", fmt.Errorf("unmarshal collected data: %w", err)
		}
	}

	// 2. Generate plan via LLM.
	plan, err := s.planner.GeneratePlan(ctx, collected)
	if err != nil {
		return "", fmt.Errorf("generate plan: %w", err)
	}

	// 3. Marshal plan to JSON.
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return "", fmt.Errorf("marshal plan: %w", err)
	}

	// 4. Update session to confirming with step 1.
	if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
		TenantID:      session.TenantID,
		Status:        statusConfirming,
		ConfirmStep:   1,
		CollectedData: session.CollectedData,
		ProposedPlan:  planJSON,
		MessageCount:  session.MessageCount,
		ChannelType:   session.ChannelType,
	}); err != nil {
		return "", fmt.Errorf("update session to confirming: %w", err)
	}

	// 5. Return formatted step 1 for review.
	return s.confirmer.FormatStep(plan, 1), nil
}

// handleConfirming processes messages during the step-by-step confirmation flow.
func (s *Service) handleConfirming(ctx context.Context, session *sqlc.OnboardingSession, text string) (string, error) {
	// 1. Unmarshal the proposed plan.
	plan := &ProposedPlan{}
	if len(session.ProposedPlan) > 0 {
		if err := json.Unmarshal(session.ProposedPlan, plan); err != nil {
			return "", fmt.Errorf("unmarshal proposed plan: %w", err)
		}
	}

	step := int(session.ConfirmStep)

	// 2. Check if the user is confirming or requesting a modification.
	if s.confirmer.IsConfirmation(text) {
		// Apply this step's configuration.
		if err := s.applyStep(ctx, session.TenantID, plan, step); err != nil {
			return "", fmt.Errorf("apply step %d: %w", step, err)
		}

		// If we're at the last step, finalize.
		if step >= totalConfirmSteps {
			return s.completeOnboarding(ctx, session)
		}

		// Advance to next step.
		nextStep := int32(step + 1)
		if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
			TenantID:      session.TenantID,
			Status:        statusConfirming,
			ConfirmStep:   nextStep,
			CollectedData: session.CollectedData,
			ProposedPlan:  session.ProposedPlan,
			MessageCount:  session.MessageCount,
			ChannelType:   session.ChannelType,
		}); err != nil {
			return "", fmt.Errorf("advance to step %d: %w", nextStep, err)
		}

		return s.confirmer.FormatStep(plan, int(nextStep)), nil
	}

	// 3. Handle modification request.
	updatedPlan, formatted, err := s.confirmer.HandleModification(ctx, plan, step, text)
	if err != nil {
		return "", fmt.Errorf("handle modification: %w", err)
	}

	// 4. Save updated plan to session.
	planJSON, err := json.Marshal(updatedPlan)
	if err != nil {
		return "", fmt.Errorf("marshal updated plan: %w", err)
	}

	if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
		TenantID:      session.TenantID,
		Status:        statusConfirming,
		ConfirmStep:   session.ConfirmStep,
		CollectedData: session.CollectedData,
		ProposedPlan:  planJSON,
		MessageCount:  session.MessageCount,
		ChannelType:   session.ChannelType,
	}); err != nil {
		return "", fmt.Errorf("update session after modification: %w", err)
	}

	return formatted, nil
}

// applyStep writes the configuration for a specific confirmation step to the database.
func (s *Service) applyStep(ctx context.Context, tenantID pgtype.UUID, plan *ProposedPlan, step int) error {
	switch step {
	case 1:
		// Step 1: Mentor & Board — write mentor/board config to organization.
		return s.applyMentorAndBoard(ctx, tenantID, plan)
	case 2:
		// Step 2: Org Structure — write org units.
		return s.applyOrgStructure(ctx, tenantID, plan)
	case 3:
		// Step 3: Policies — write framework/policies.
		return s.applyPolicies(ctx, tenantID, plan)
	case 4:
		// Step 4: Schedule — write schedule/comm config and finalize organization.
		return s.applySchedule(ctx, tenantID, plan)
	default:
		return fmt.Errorf("unknown step %d", step)
	}
}

// applyMentorAndBoard writes mentor and board configuration.
func (s *Service) applyMentorAndBoard(ctx context.Context, tenantID pgtype.UUID, plan *ProposedPlan) error {
	// Encode board as team_structure JSON.
	boardJSON, err := json.Marshal(plan.Board)
	if err != nil {
		return fmt.Errorf("marshal board: %w", err)
	}

	return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
		TenantID:      tenantID,
		TeamStructure: boardJSON,
		// Other fields set to zero-value (NULL).
		Industry:             pgtype.Text{},
		Size:                 pgtype.Int4{},
		Stage:                pgtype.Text{},
		BusinessModel:        pgtype.Text{},
		ManagementPainPoints: nil,
		CurrentProjects:      nil,
		TargetFramework:      pgtype.Text{},
		CommunicationTools:   nil,
		CulturePreferences:   nil,
	})
}

// applyOrgStructure deletes existing org units and creates new ones from the plan.
func (s *Service) applyOrgStructure(ctx context.Context, tenantID pgtype.UUID, plan *ProposedPlan) error {
	// Delete existing units.
	if err := s.db.DeleteOrgUnitsByTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("delete existing org units: %w", err)
	}

	// Build a map from ref_id -> created UUID for parent linking.
	refToID := make(map[string]pgtype.UUID)

	// Create units in order (parents first, then children).
	// First pass: root units (no parent).
	for i, unit := range plan.OrgDesign.Units {
		if unit.ParentRefID != "" {
			continue
		}
		created, err := s.db.CreateOrgUnit(ctx, sqlc.CreateOrgUnitParams{
			TenantID:         tenantID,
			ParentID:         pgtype.UUID{}, // no parent
			Name:             unit.Name,
			UnitType:         unit.UnitType,
			HeadRole:         pgtype.Text{String: unit.HeadRole, Valid: unit.HeadRole != ""},
			Responsibilities: pgtype.Text{String: unit.Responsibilities, Valid: unit.Responsibilities != ""},
			SortOrder:        int32(i),
		})
		if err != nil {
			return fmt.Errorf("create org unit %q: %w", unit.Name, err)
		}
		refToID[unit.RefID] = created.ID
	}

	// Second pass: child units (have a parent).
	for i, unit := range plan.OrgDesign.Units {
		if unit.ParentRefID == "" {
			continue
		}
		parentID := refToID[unit.ParentRefID]
		created, err := s.db.CreateOrgUnit(ctx, sqlc.CreateOrgUnitParams{
			TenantID:         tenantID,
			ParentID:         parentID,
			Name:             unit.Name,
			UnitType:         unit.UnitType,
			HeadRole:         pgtype.Text{String: unit.HeadRole, Valid: unit.HeadRole != ""},
			Responsibilities: pgtype.Text{String: unit.Responsibilities, Valid: unit.Responsibilities != ""},
			SortOrder:        int32(i),
		})
		if err != nil {
			return fmt.Errorf("create org unit %q: %w", unit.Name, err)
		}
		refToID[unit.RefID] = created.ID
	}

	return nil
}

// applyPolicies writes framework and policy configuration.
func (s *Service) applyPolicies(ctx context.Context, tenantID pgtype.UUID, plan *ProposedPlan) error {
	return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
		TenantID:        tenantID,
		TargetFramework: pgtype.Text{String: plan.Policies.Framework, Valid: plan.Policies.Framework != ""},
		// Other fields set to zero-value (NULL).
		Industry:             pgtype.Text{},
		Size:                 pgtype.Int4{},
		Stage:                pgtype.Text{},
		BusinessModel:        pgtype.Text{},
		ManagementPainPoints: nil,
		CurrentProjects:      nil,
		TeamStructure:        nil,
		CommunicationTools:   nil,
		CulturePreferences:   nil,
	})
}

// applySchedule writes schedule and communication config, and fills in all
// remaining organization fields from collected data.
func (s *Service) applySchedule(ctx context.Context, tenantID pgtype.UUID, plan *ProposedPlan) error {
	// At step 4, we write everything that hasn't been written yet.
	// Communication tools come from collected data, but we don't have
	// direct access here. Schedule/comm fields go to culture_preferences.
	scheduleJSON, err := json.Marshal(plan.Schedule)
	if err != nil {
		return fmt.Errorf("marshal schedule: %w", err)
	}

	return s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
		TenantID:           tenantID,
		CulturePreferences: scheduleJSON,
		// Other fields set to zero-value (NULL).
		Industry:             pgtype.Text{},
		Size:                 pgtype.Int4{},
		Stage:                pgtype.Text{},
		BusinessModel:        pgtype.Text{},
		ManagementPainPoints: nil,
		CurrentProjects:      nil,
		TargetFramework:      pgtype.Text{},
		TeamStructure:        nil,
		CommunicationTools:   nil,
	})
}

// completeOnboarding writes all collected data to the organization, marks the
// tenant onboarding as completed, and cleans up the session.
func (s *Service) completeOnboarding(ctx context.Context, session *sqlc.OnboardingSession) (string, error) {
	// Unmarshal collected data for the final organization update.
	collected := &CollectedData{}
	if len(session.CollectedData) > 0 {
		if err := json.Unmarshal(session.CollectedData, collected); err != nil {
			return "", fmt.Errorf("unmarshal collected data for completion: %w", err)
		}
	}

	// Unmarshal plan for full data.
	plan := &ProposedPlan{}
	if len(session.ProposedPlan) > 0 {
		if err := json.Unmarshal(session.ProposedPlan, plan); err != nil {
			return "", fmt.Errorf("unmarshal plan for completion: %w", err)
		}
	}

	// Write comprehensive organization data.
	projectsJSON, _ := json.Marshal(collected.CurrentProjects)
	boardJSON, _ := json.Marshal(plan.Board)
	cultureJSON, _ := json.Marshal(map[string]interface{}{
		"preferences": collected.CulturePrefs,
		"schedule":    plan.Schedule,
		"policies":    plan.Policies,
	})

	if err := s.db.UpdateOrganizationFromOnboarding(ctx, sqlc.UpdateOrganizationFromOnboardingParams{
		TenantID:             session.TenantID,
		Industry:             pgtype.Text{String: collected.Industry, Valid: collected.Industry != ""},
		Size:                 pgtype.Int4{Int32: int32(collected.TeamSize), Valid: collected.TeamSize > 0},
		Stage:                pgtype.Text{String: collected.CompanyStage, Valid: collected.CompanyStage != ""},
		BusinessModel:        pgtype.Text{String: collected.BusinessModel, Valid: collected.BusinessModel != ""},
		ManagementPainPoints: collected.PainPoints,
		CurrentProjects:      projectsJSON,
		TargetFramework:      pgtype.Text{String: plan.Policies.Framework, Valid: plan.Policies.Framework != ""},
		TeamStructure:        boardJSON,
		CommunicationTools:   collected.CommTools,
		CulturePreferences:   cultureJSON,
	}); err != nil {
		return "", fmt.Errorf("update organization: %w", err)
	}

	// Mark tenant onboarding as completed.
	if err := s.db.SetTenantOnboardingCompleted(ctx, session.TenantID); err != nil {
		return "", fmt.Errorf("set onboarding completed: %w", err)
	}

	// Update session status to active.
	if err := s.db.UpdateOnboardingSession(ctx, sqlc.UpdateOnboardingSessionParams{
		TenantID:      session.TenantID,
		Status:        statusActive,
		ConfirmStep:   totalConfirmSteps,
		CollectedData: session.CollectedData,
		ProposedPlan:  session.ProposedPlan,
		MessageCount:  session.MessageCount,
		ChannelType:   session.ChannelType,
	}); err != nil {
		return "", fmt.Errorf("update session to active: %w", err)
	}

	// Clean up chat history from Redis.
	historyKey := fmt.Sprintf("onboarding:history:%x", session.TenantID.Bytes)
	s.redis.Del(ctx, historyKey)

	return "Onboarding complete! Your AI management system is now configured and ready to go. " +
		"You can start inviting team members and I'll begin daily check-ins according to the schedule we set up.", nil
}

// historyKey returns the Redis key for chat history.
func historyKey(tenantID pgtype.UUID) string {
	return fmt.Sprintf("onboarding:history:%x", tenantID.Bytes)
}

// loadChatHistory loads the conversation history from Redis.
func (s *Service) loadChatHistory(ctx context.Context, tenantID pgtype.UUID) []brain.ChatMessage {
	key := historyKey(tenantID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var history []brain.ChatMessage
	if err := json.Unmarshal([]byte(data), &history); err != nil {
		slog.Warn("failed to unmarshal chat history", "error", err)
		return nil
	}
	return history
}

// saveChatHistory saves the conversation history to Redis with a TTL.
func (s *Service) saveChatHistory(ctx context.Context, tenantID pgtype.UUID, history []brain.ChatMessage) {
	key := historyKey(tenantID)
	data, err := json.Marshal(history)
	if err != nil {
		slog.Warn("failed to marshal chat history", "error", err)
		return
	}
	if err := s.redis.Set(ctx, key, data, historyTTL).Err(); err != nil {
		slog.Warn("failed to save chat history", "error", err)
	}
}
