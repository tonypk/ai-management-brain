package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ---------------------------------------------------------------------------
// Mock: Database (Querier)
// ---------------------------------------------------------------------------

type mockServiceDB struct {
	getSession     func(ctx context.Context, tenantID pgtype.UUID) (sqlc.OnboardingSession, error)
	createSession  func(ctx context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error)
	updateSession  func(ctx context.Context, arg sqlc.UpdateOnboardingSessionParams) error
	deleteSession  func(ctx context.Context, tenantID pgtype.UUID) error
	updateOrg      func(ctx context.Context, arg sqlc.UpdateOrganizationFromOnboardingParams) error
	setCompleted   func(ctx context.Context, id pgtype.UUID) error
	createOrgUnit  func(ctx context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error)
	deleteOrgUnits func(ctx context.Context, tenantID pgtype.UUID) error
}

func (m *mockServiceDB) GetOnboardingSession(ctx context.Context, tenantID pgtype.UUID) (sqlc.OnboardingSession, error) {
	if m.getSession != nil {
		return m.getSession(ctx, tenantID)
	}
	return sqlc.OnboardingSession{}, errors.New("not found")
}

func (m *mockServiceDB) CreateOnboardingSession(ctx context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error) {
	if m.createSession != nil {
		return m.createSession(ctx, arg)
	}
	return sqlc.OnboardingSession{
		TenantID:    arg.TenantID,
		Status:      "onboarding",
		ChannelType: arg.ChannelType,
	}, nil
}

func (m *mockServiceDB) UpdateOnboardingSession(ctx context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
	if m.updateSession != nil {
		return m.updateSession(ctx, arg)
	}
	return nil
}

func (m *mockServiceDB) DeleteOnboardingSession(ctx context.Context, tenantID pgtype.UUID) error {
	if m.deleteSession != nil {
		return m.deleteSession(ctx, tenantID)
	}
	return nil
}

func (m *mockServiceDB) UpdateOrganizationFromOnboarding(ctx context.Context, arg sqlc.UpdateOrganizationFromOnboardingParams) error {
	if m.updateOrg != nil {
		return m.updateOrg(ctx, arg)
	}
	return nil
}

func (m *mockServiceDB) SetTenantOnboardingCompleted(ctx context.Context, id pgtype.UUID) error {
	if m.setCompleted != nil {
		return m.setCompleted(ctx, id)
	}
	return nil
}

func (m *mockServiceDB) CreateOrgUnit(ctx context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error) {
	if m.createOrgUnit != nil {
		return m.createOrgUnit(ctx, arg)
	}
	return sqlc.OrgUnit{
		ID:       pgtype.UUID{Bytes: [16]byte{99}, Valid: true},
		TenantID: arg.TenantID,
		Name:     arg.Name,
		UnitType: arg.UnitType,
	}, nil
}

func (m *mockServiceDB) DeleteOrgUnitsByTenant(ctx context.Context, tenantID pgtype.UUID) error {
	if m.deleteOrgUnits != nil {
		return m.deleteOrgUnits(ctx, tenantID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: Redis
// ---------------------------------------------------------------------------

type mockServiceRedis struct {
	locked bool
	data   map[string]string
}

func newMockRedis() *mockServiceRedis {
	return &mockServiceRedis{data: make(map[string]string)}
}

func (m *mockServiceRedis) SetNX(_ context.Context, key string, value interface{}, _ time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	if m.locked {
		cmd.SetVal(false)
	} else {
		m.locked = true
		m.data[key] = "1"
		cmd.SetVal(true)
	}
	return cmd
}

func (m *mockServiceRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	for _, k := range keys {
		delete(m.data, k)
	}
	m.locked = false
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (m *mockServiceRedis) Get(_ context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	if val, ok := m.data[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *mockServiceRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	switch v := value.(type) {
	case string:
		m.data[key] = v
	case []byte:
		m.data[key] = string(v)
	default:
		b, _ := json.Marshal(v)
		m.data[key] = string(b)
	}
	cmd.SetVal("OK")
	return cmd
}

// ---------------------------------------------------------------------------
// Mock: LLM (single-turn, used for extraction/planning/confirmation)
// ---------------------------------------------------------------------------

type mockServiceLLM struct {
	responses []string
	errs      []error
	callCount int
}

func (m *mockServiceLLM) Chat(_ context.Context, _, _ string) (string, error) {
	idx := m.callCount
	m.callCount++
	if idx < len(m.errs) && m.errs[idx] != nil {
		return "", m.errs[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", errors.New("no more mock responses")
}

// ---------------------------------------------------------------------------
// Mock: ChatLLMClient (multi-turn, used for onboarding dialogue)
// ---------------------------------------------------------------------------

type mockServiceChatLLM struct {
	response string
	err      error
}

func (m *mockServiceChatLLM) ChatWithHistory(_ context.Context, _ string, _ []brain.ChatMessage, _ string) (string, error) {
	return m.response, m.err
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

var testTenantID = pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4}, Valid: true}

func fullCollectedData() *CollectedData {
	return &CollectedData{
		Industry:        "SaaS",
		CompanyStage:    "Series A",
		BusinessModel:   "B2B",
		TeamSize:        25,
		OrgStructure:    "Flat",
		CurrentProjects: "Platform v2",
		PainPoints:      []string{"communication", "deadlines"},
		CommTools:       []string{"Slack", "Zoom"},
		CulturePrefs:    "Collaborative",
		GoalFramework:   "OKR",
	}
}

func testPlan() *ProposedPlan {
	return &ProposedPlan{
		Mentor: MentorPlan{
			PrimaryID:   "musk",
			SecondaryID: "inamori",
			BlendWeight: 0.7,
			Reasoning:   "test",
		},
		Board: []SeatPlan{
			{SeatType: "ceo", PersonaID: "musk", Reasoning: "test"},
			{SeatType: "cto", PersonaID: "grove", Reasoning: "test"},
		},
		OrgDesign: OrgDesignPlan{
			Units: []OrgUnitPlan{
				{RefID: "ceo", ParentRefID: "", Name: "CEO Office", UnitType: "department", HeadRole: "CEO"},
				{RefID: "eng", ParentRefID: "ceo", Name: "Engineering", UnitType: "department", HeadRole: "VP Eng"},
			},
			Reasoning: "test",
		},
		Policies: PolicyPlan{
			Framework:        "okr",
			CheckinQuestions: []string{"What did you do today?"},
			TrackingFocus:    []string{"velocity"},
			RiskRules: RiskRules{
				ConsecutiveMisses:      3,
				SentimentDropThreshold: -0.3,
				UrgentKeywords:         []string{"urgent"},
			},
			Cadence: Cadence{
				DailyActions:   []string{"checkin"},
				WeeklyActions:  []string{"review"},
				WeeklyDay:      "friday",
				MonthlyActions: []string{"retro"},
				MonthlyDay:     1,
			},
			Reasoning: "test",
		},
		Schedule: SchedulePlan{
			Checkin:    "0 9 * * 1-5",
			Chase:      "30 17 * * 1-5",
			Summary:    "0 19 * * 1-5",
			Briefing:   "0 8 * * 1-5",
			SignalScan: "*/30 9-18 * * 1-5",
			Timezone:   "Asia/Manila",
		},
		Reasoning: "test plan",
	}
}

func testPlanJSON() []byte {
	data, _ := json.Marshal(testPlan())
	return data
}

func fullCollectedJSON() []byte {
	data, _ := json.Marshal(fullCollectedData())
	return data
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandleMessage_OnboardingState_DialogueContinues(t *testing.T) {
	// Scenario: User sends a message during onboarding, not enough info collected yet.
	// Expected: Extraction runs, chat response returned, session updated.
	partialData := &CollectedData{Industry: "SaaS"}
	partialJSON, _ := json.Marshal(partialData)

	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "onboarding",
				ConfirmStep:   0,
				CollectedData: partialJSON,
				MessageCount:  1,
				ChannelType:   "telegram",
			}, nil
		},
	}

	rds := newMockRedis()

	// Extractor returns same data (no new info from this message).
	extractLLM := &mockServiceLLM{responses: []string{`{}`}, errs: []error{nil}}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{response: "Tell me more about your team structure."}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "Hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Tell me more about your team structure." {
		t.Errorf("unexpected reply: %q", reply)
	}
}

func TestHandleMessage_OnboardingToConfiguringTransition(t *testing.T) {
	// Scenario: The user's message completes all required fields.
	// Expected: Extraction fills everything -> transitions to configuring -> generates plan -> returns step 1.
	partialData := &CollectedData{
		Industry:        "SaaS",
		CompanyStage:    "Series A",
		BusinessModel:   "B2B",
		TeamSize:        25,
		OrgStructure:    "Flat",
		CurrentProjects: "Platform v2",
		PainPoints:      []string{"communication"},
		// Missing: CommTools (the extraction will fill it)
	}
	partialJSON, _ := json.Marshal(partialData)

	var updatedStatuses []string
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "onboarding",
				ConfirmStep:   0,
				CollectedData: partialJSON,
				MessageCount:  5,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			updatedStatuses = append(updatedStatuses, arg.Status)
			return nil
		},
	}

	rds := newMockRedis()

	// Extraction LLM returns the missing CommTools.
	extractLLM := &mockServiceLLM{
		responses: []string{`{"comm_tools":["Slack","Zoom"]}`},
		errs:      []error{nil},
	}

	// Plan LLM returns a valid plan.
	planLLM := &mockServiceLLM{
		responses: []string{string(testPlanJSON())},
		errs:      []error{nil},
	}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{} // should not be called

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "We use Slack and Zoom")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reply should be step 1 formatted output.
	if !strings.Contains(reply, "Step 1") {
		t.Errorf("expected step 1 in reply, got: %q", reply)
	}
	if !strings.Contains(reply, "musk") {
		t.Errorf("expected mentor 'musk' in reply, got: %q", reply)
	}

	// Verify status transitions: first to configuring, then to confirming.
	if len(updatedStatuses) < 2 {
		t.Fatalf("expected at least 2 session updates, got %d", len(updatedStatuses))
	}
	if updatedStatuses[0] != "configuring" {
		t.Errorf("expected first update to 'configuring', got %q", updatedStatuses[0])
	}
	if updatedStatuses[1] != "confirming" {
		t.Errorf("expected second update to 'confirming', got %q", updatedStatuses[1])
	}
}

func TestHandleMessage_ConfiguringState(t *testing.T) {
	// Scenario: Session is already in configuring state.
	// Expected: Generates plan, transitions to confirming, returns step 1.
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "configuring",
				ConfirmStep:   0,
				CollectedData: fullCollectedJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
	}

	rds := newMockRedis()

	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{
		responses: []string{string(testPlanJSON())},
		errs:      []error{nil},
	}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(reply, "Step 1") {
		t.Errorf("expected step 1 format, got: %q", reply)
	}
	if !strings.Contains(reply, "Reply OK to confirm") {
		t.Errorf("expected confirmation prompt, got: %q", reply)
	}
}

func TestHandleMessage_ConfirmingStep1_OK(t *testing.T) {
	// Scenario: User confirms step 1, should apply step 1 and advance to step 2.
	var updateOrgCalled bool
	var lastUpdateStatus string
	var lastUpdateStep int32

	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "confirming",
				ConfirmStep:   1,
				CollectedData: fullCollectedJSON(),
				ProposedPlan:  testPlanJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			lastUpdateStatus = arg.Status
			lastUpdateStep = arg.ConfirmStep
			return nil
		},
		updateOrg: func(_ context.Context, _ sqlc.UpdateOrganizationFromOnboardingParams) error {
			updateOrgCalled = true
			return nil
		},
	}

	rds := newMockRedis()
	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "ok")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have applied step 1 (UpdateOrganization for mentor/board).
	if !updateOrgCalled {
		t.Error("expected UpdateOrganizationFromOnboarding to be called for step 1")
	}

	// Should advance to step 2.
	if lastUpdateStatus != "confirming" {
		t.Errorf("expected status 'confirming', got %q", lastUpdateStatus)
	}
	if lastUpdateStep != 2 {
		t.Errorf("expected step 2, got %d", lastUpdateStep)
	}

	// Reply should be step 2 formatted output.
	if !strings.Contains(reply, "Step 2") {
		t.Errorf("expected step 2 in reply, got: %q", reply)
	}
}

func TestHandleMessage_ConfirmingStep4_CompletesOnboarding(t *testing.T) {
	// Scenario: User confirms step 4 (final step), should complete onboarding.
	var completedCalled bool
	var finalStatus string

	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "confirming",
				ConfirmStep:   4,
				CollectedData: fullCollectedJSON(),
				ProposedPlan:  testPlanJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			finalStatus = arg.Status
			return nil
		},
		updateOrg: func(_ context.Context, _ sqlc.UpdateOrganizationFromOnboardingParams) error {
			return nil
		},
		setCompleted: func(_ context.Context, _ pgtype.UUID) error {
			completedCalled = true
			return nil
		},
	}

	rds := newMockRedis()
	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "ok")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !completedCalled {
		t.Error("expected SetTenantOnboardingCompleted to be called")
	}
	if finalStatus != "active" {
		t.Errorf("expected final status 'active', got %q", finalStatus)
	}

	// Reply should indicate completion.
	if !strings.Contains(reply, "complete") && !strings.Contains(reply, "Complete") {
		t.Errorf("expected completion message, got: %q", reply)
	}
}

func TestHandleMessage_ConcurrencyLock(t *testing.T) {
	// Scenario: Second message arrives while first is still being processed.
	// Expected: Second message gets "thinking" response.
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "onboarding",
				CollectedData: []byte("{}"),
				ChannelType:   "telegram",
			}, nil
		},
	}

	rds := newMockRedis()

	extractLLM := &mockServiceLLM{responses: []string{`{}`}, errs: []error{nil}}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{response: "What industry?"}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// Simulate lock already held.
	rds.locked = true

	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(reply, "still thinking") {
		t.Errorf("expected 'still thinking' message, got: %q", reply)
	}
}

func TestHandleMessage_ModificationInConfirming(t *testing.T) {
	// Scenario: User requests a modification instead of confirming in step 1.
	modifiedPlan := testPlan()
	modifiedPlan.Mentor.PrimaryID = "inamori"
	modifiedPlanJSON, _ := json.Marshal(modifiedPlan)

	var savedPlan []byte
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "confirming",
				ConfirmStep:   1,
				CollectedData: fullCollectedJSON(),
				ProposedPlan:  testPlanJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			savedPlan = arg.ProposedPlan
			return nil
		},
	}

	rds := newMockRedis()
	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{}
	// The confirmer LLM returns the modified plan.
	confirmLLM := &mockServiceLLM{
		responses: []string{string(modifiedPlanJSON)},
		errs:      []error{nil},
	}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "change mentor to inamori")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reply should contain the modified mentor.
	if !strings.Contains(reply, "inamori") {
		t.Errorf("expected 'inamori' in reply, got: %q", reply)
	}

	// Verify the saved plan was updated.
	if savedPlan == nil {
		t.Fatal("expected plan to be saved")
	}
	var saved ProposedPlan
	if err := json.Unmarshal(savedPlan, &saved); err != nil {
		t.Fatalf("failed to unmarshal saved plan: %v", err)
	}
	if saved.Mentor.PrimaryID != "inamori" {
		t.Errorf("expected saved mentor 'inamori', got %q", saved.Mentor.PrimaryID)
	}
}

func TestHandleMessage_AlreadyComplete(t *testing.T) {
	// Scenario: Session status is "active" (already completed).
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID: testTenantID,
				Status:   "active",
			}, nil
		},
	}

	rds := newMockRedis()
	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(reply, "already complete") {
		t.Errorf("expected 'already complete' message, got: %q", reply)
	}
}

func TestHandleMessage_NewSession(t *testing.T) {
	// Scenario: No session exists, so a new one is created.
	var createdChannel string
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{}, errors.New("not found")
		},
		createSession: func(_ context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error) {
			createdChannel = arg.ChannelType
			return sqlc.OnboardingSession{
				TenantID:    arg.TenantID,
				Status:      "onboarding",
				ChannelType: arg.ChannelType,
			}, nil
		},
	}

	rds := newMockRedis()
	extractLLM := &mockServiceLLM{responses: []string{`{}`}, errs: []error{nil}}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{response: "Welcome! What industry is your company in?"}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	reply, err := svc.HandleMessage(context.Background(), testTenantID, "slack", "user1", "hi")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdChannel != "slack" {
		t.Errorf("expected channel 'slack', got %q", createdChannel)
	}
	if !strings.Contains(reply, "Welcome") {
		t.Errorf("expected welcome message, got: %q", reply)
	}
}

func TestHandleMessage_ChatHistoryPersisted(t *testing.T) {
	// Scenario: Verify that chat history is saved and loaded from Redis.
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "onboarding",
				CollectedData: []byte("{}"),
				MessageCount:  0,
				ChannelType:   "telegram",
			}, nil
		},
	}

	rds := newMockRedis()

	// Pre-populate history.
	priorHistory := []brain.ChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "What industry?"},
	}
	historyJSON, _ := json.Marshal(priorHistory)
	rds.data[historyKey(testTenantID)] = string(historyJSON)

	extractLLM := &mockServiceLLM{responses: []string{`{}`}, errs: []error{nil}}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{response: "How many employees?"}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	_, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "We are in SaaS")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify updated history was saved back.
	key := historyKey(testTenantID)
	savedData, ok := rds.data[key]
	if !ok {
		t.Fatal("expected chat history to be saved in Redis")
	}

	var savedHistory []brain.ChatMessage
	if err := json.Unmarshal([]byte(savedData), &savedHistory); err != nil {
		t.Fatalf("failed to unmarshal saved history: %v", err)
	}

	// Should have 4 messages: 2 prior + 1 user + 1 assistant.
	if len(savedHistory) != 4 {
		t.Fatalf("expected 4 messages in history, got %d", len(savedHistory))
	}
	if savedHistory[2].Role != "user" || savedHistory[2].Content != "We are in SaaS" {
		t.Errorf("unexpected 3rd message: %+v", savedHistory[2])
	}
	if savedHistory[3].Role != "assistant" || savedHistory[3].Content != "How many employees?" {
		t.Errorf("unexpected 4th message: %+v", savedHistory[3])
	}
}

func TestApplyStep_OrgStructure(t *testing.T) {
	// Verify that applyStep(2) deletes old units and creates new ones.
	var deletedTenant pgtype.UUID
	var createdUnits []string

	db := &mockServiceDB{
		deleteOrgUnits: func(_ context.Context, tenantID pgtype.UUID) error {
			deletedTenant = tenantID
			return nil
		},
		createOrgUnit: func(_ context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error) {
			createdUnits = append(createdUnits, arg.Name)
			return sqlc.OrgUnit{
				ID:       pgtype.UUID{Bytes: [16]byte{byte(len(createdUnits))}, Valid: true},
				TenantID: arg.TenantID,
				Name:     arg.Name,
			}, nil
		},
	}

	rds := newMockRedis()
	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	plan := testPlan()
	err := svc.applyStep(context.Background(), testTenantID, plan, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deletedTenant != testTenantID {
		t.Error("expected DeleteOrgUnitsByTenant to be called with test tenant ID")
	}

	// Plan has 2 units: CEO Office (root), Engineering (child).
	if len(createdUnits) != 2 {
		t.Fatalf("expected 2 org units created, got %d: %v", len(createdUnits), createdUnits)
	}
	if createdUnits[0] != "CEO Office" {
		t.Errorf("expected first unit 'CEO Office', got %q", createdUnits[0])
	}
	if createdUnits[1] != "Engineering" {
		t.Errorf("expected second unit 'Engineering', got %q", createdUnits[1])
	}
}

func TestNewService(t *testing.T) {
	db := &mockServiceDB{}
	rds := newMockRedis()
	extractLLM := &mockServiceLLM{}
	planLLM := &mockServiceLLM{}
	confirmLLM := &mockServiceLLM{}
	chatLLM := &mockServiceChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.db != db {
		t.Error("expected service to hold the provided DB")
	}
	if svc.redis != rds {
		t.Error("expected service to hold the provided Redis client")
	}
	if svc.extractor == nil {
		t.Error("expected service to have an extractor")
	}
	if svc.planner == nil {
		t.Error("expected service to have a planner")
	}
	if svc.confirmer == nil {
		t.Error("expected service to have a confirmer")
	}
	if svc.chatLLM != chatLLM {
		t.Error("expected service to hold the provided chat LLM")
	}
}

func TestConfirmingStep2_OK_AdvancesToStep3(t *testing.T) {
	// Scenario: User confirms step 2 (org structure).
	var deletedOrgUnits bool
	var createdUnits int
	var lastStep int32

	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "confirming",
				ConfirmStep:   2,
				CollectedData: fullCollectedJSON(),
				ProposedPlan:  testPlanJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			lastStep = arg.ConfirmStep
			return nil
		},
		deleteOrgUnits: func(_ context.Context, _ pgtype.UUID) error {
			deletedOrgUnits = true
			return nil
		},
		createOrgUnit: func(_ context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error) {
			createdUnits++
			return sqlc.OrgUnit{
				ID:       pgtype.UUID{Bytes: [16]byte{byte(createdUnits)}, Valid: true},
				TenantID: arg.TenantID,
				Name:     arg.Name,
			}, nil
		},
	}

	rds := newMockRedis()
	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !deletedOrgUnits {
		t.Error("expected org units to be deleted before recreation")
	}
	if createdUnits != 2 {
		t.Errorf("expected 2 org units created, got %d", createdUnits)
	}
	if lastStep != 3 {
		t.Errorf("expected advance to step 3, got step %d", lastStep)
	}
	if !strings.Contains(reply, "Step 3") {
		t.Errorf("expected step 3 in reply, got: %q", reply)
	}
}

func TestConfirmingStep3_OK_AdvancesToStep4(t *testing.T) {
	// Scenario: User confirms step 3 (policies).
	var updateOrgCalled bool
	var lastStep int32

	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID:      testTenantID,
				Status:        "confirming",
				ConfirmStep:   3,
				CollectedData: fullCollectedJSON(),
				ProposedPlan:  testPlanJSON(),
				MessageCount:  10,
				ChannelType:   "telegram",
			}, nil
		},
		updateSession: func(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
			lastStep = arg.ConfirmStep
			return nil
		},
		updateOrg: func(_ context.Context, _ sqlc.UpdateOrganizationFromOnboardingParams) error {
			updateOrgCalled = true
			return nil
		},
	}

	rds := newMockRedis()
	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	reply, err := svc.HandleMessage(context.Background(), testTenantID, "telegram", "user1", "confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !updateOrgCalled {
		t.Error("expected UpdateOrganizationFromOnboarding to be called for step 3")
	}
	if lastStep != 4 {
		t.Errorf("expected advance to step 4, got step %d", lastStep)
	}
	if !strings.Contains(reply, "Step 4") {
		t.Errorf("expected step 4 in reply, got: %q", reply)
	}
}

func TestGetOrCreateSession_ExistingSession(t *testing.T) {
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{
				TenantID: testTenantID,
				Status:   "onboarding",
			}, nil
		},
	}

	rds := newMockRedis()
	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	session, err := svc.getOrCreateSession(context.Background(), testTenantID, "telegram")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != "onboarding" {
		t.Errorf("expected status 'onboarding', got %q", session.Status)
	}
}

func TestGetOrCreateSession_NewSession(t *testing.T) {
	var createCalled bool
	db := &mockServiceDB{
		getSession: func(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
			return sqlc.OnboardingSession{}, errors.New("not found")
		},
		createSession: func(_ context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error) {
			createCalled = true
			return sqlc.OnboardingSession{
				TenantID:    arg.TenantID,
				Status:      "onboarding",
				ChannelType: arg.ChannelType,
			}, nil
		},
	}

	rds := newMockRedis()
	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	session, err := svc.getOrCreateSession(context.Background(), testTenantID, "slack")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Error("expected CreateOnboardingSession to be called")
	}
	if session.ChannelType != "slack" {
		t.Errorf("expected channel 'slack', got %q", session.ChannelType)
	}
}

func TestCompleteOnboarding_CleansUpHistory(t *testing.T) {
	// Verify that completeOnboarding removes chat history from Redis.
	db := &mockServiceDB{}
	rds := newMockRedis()

	// Pre-populate Redis with history.
	key := historyKey(testTenantID)
	rds.data[key] = `[{"role":"user","content":"hello"}]`

	svc := NewService(db, rds, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceLLM{}, &mockServiceChatLLM{})

	session := &sqlc.OnboardingSession{
		TenantID:      testTenantID,
		Status:        "confirming",
		ConfirmStep:   4,
		CollectedData: fullCollectedJSON(),
		ProposedPlan:  testPlanJSON(),
		ChannelType:   "telegram",
	}

	reply, err := svc.completeOnboarding(context.Background(), session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(reply, "complete") {
		t.Errorf("expected completion message, got: %q", reply)
	}

	// History should be cleaned up.
	if _, exists := rds.data[key]; exists {
		t.Error("expected chat history to be deleted from Redis after completion")
	}
}
