package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/tonypk/ai-management-brain/internal/brain"
	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// ---------------------------------------------------------------------------
// Integration mock: Stateful Database
// ---------------------------------------------------------------------------

// integrationMockDB is a stateful mock that tracks the onboarding session
// across multiple HandleMessage calls, simulating a real database.
type integrationMockDB struct {
	mu sync.Mutex

	// Session state — updated by Create/Update, returned by Get.
	session          *sqlc.OnboardingSession
	sessionExists    bool
	tenantCompleted  bool
	orgUnitsDeleted  bool
	orgUnitsCreated  []string
	updateOrgCalls   int
	orgUnitIDCounter int
}

func (m *integrationMockDB) GetOnboardingSession(_ context.Context, _ pgtype.UUID) (sqlc.OnboardingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.sessionExists || m.session == nil {
		return sqlc.OnboardingSession{}, errors.New("not found")
	}
	return *m.session, nil
}

func (m *integrationMockDB) CreateOnboardingSession(_ context.Context, arg sqlc.CreateOnboardingSessionParams) (sqlc.OnboardingSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := sqlc.OnboardingSession{
		ID:          pgtype.UUID{Bytes: [16]byte{0xAA}, Valid: true},
		TenantID:    arg.TenantID,
		Status:      "onboarding",
		ConfirmStep: 0,
		ChannelType: arg.ChannelType,
	}
	m.session = &s
	m.sessionExists = true
	return s, nil
}

func (m *integrationMockDB) UpdateOnboardingSession(_ context.Context, arg sqlc.UpdateOnboardingSessionParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return errors.New("no session to update")
	}
	m.session.Status = arg.Status
	m.session.ConfirmStep = arg.ConfirmStep
	m.session.CollectedData = arg.CollectedData
	m.session.ProposedPlan = arg.ProposedPlan
	m.session.MessageCount = arg.MessageCount
	m.session.ChannelType = arg.ChannelType
	return nil
}

func (m *integrationMockDB) DeleteOnboardingSession(_ context.Context, _ pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.session = nil
	m.sessionExists = false
	return nil
}

func (m *integrationMockDB) UpdateOrganizationFromOnboarding(_ context.Context, _ sqlc.UpdateOrganizationFromOnboardingParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateOrgCalls++
	return nil
}

func (m *integrationMockDB) SetTenantOnboardingCompleted(_ context.Context, _ pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenantCompleted = true
	return nil
}

func (m *integrationMockDB) CreateOrgUnit(_ context.Context, arg sqlc.CreateOrgUnitParams) (sqlc.OrgUnit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orgUnitIDCounter++
	m.orgUnitsCreated = append(m.orgUnitsCreated, arg.Name)
	return sqlc.OrgUnit{
		ID:       pgtype.UUID{Bytes: [16]byte{byte(m.orgUnitIDCounter)}, Valid: true},
		TenantID: arg.TenantID,
		Name:     arg.Name,
		UnitType: arg.UnitType,
	}, nil
}

func (m *integrationMockDB) DeleteOrgUnitsByTenant(_ context.Context, _ pgtype.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orgUnitsDeleted = true
	return nil
}

// ---------------------------------------------------------------------------
// Integration mock: Redis (in-memory with per-key locking)
// ---------------------------------------------------------------------------

type integrationMockRedis struct {
	mu   sync.Mutex
	data map[string]string
}

func newIntegrationRedis() *integrationMockRedis {
	return &integrationMockRedis{data: make(map[string]string)}
}

func (m *integrationMockRedis) SetNX(_ context.Context, key string, _ interface{}, _ time.Duration) *redis.BoolCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := redis.NewBoolCmd(context.Background())
	if _, exists := m.data[key]; exists {
		cmd.SetVal(false)
	} else {
		m.data[key] = "1"
		cmd.SetVal(true)
	}
	return cmd
}

func (m *integrationMockRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := redis.NewIntCmd(context.Background())
	for _, k := range keys {
		delete(m.data, k)
	}
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (m *integrationMockRedis) Get(_ context.Context, key string) *redis.StringCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := redis.NewStringCmd(context.Background())
	if val, ok := m.data[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *integrationMockRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	m.mu.Lock()
	defer m.mu.Unlock()
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
// Integration mock: LLM (scripted sequence of responses)
// ---------------------------------------------------------------------------

type integrationMockLLM struct {
	mu        sync.Mutex
	responses []string
	callCount int
}

func (m *integrationMockLLM) Chat(_ context.Context, _, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.callCount
	m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", fmt.Errorf("integrationMockLLM: no response at index %d (have %d)", idx, len(m.responses))
}

// ---------------------------------------------------------------------------
// Integration mock: ChatLLM (scripted sequence of responses)
// ---------------------------------------------------------------------------

type integrationMockChatLLM struct {
	mu        sync.Mutex
	responses []string
	callCount int
}

func (m *integrationMockChatLLM) ChatWithHistory(_ context.Context, _ string, _ []brain.ChatMessage, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.callCount
	m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", fmt.Errorf("integrationMockChatLLM: no response at index %d (have %d)", idx, len(m.responses))
}

// ---------------------------------------------------------------------------
// Helper: build a valid ProposedPlan for integration tests
// ---------------------------------------------------------------------------

func integrationPlan() *ProposedPlan {
	return &ProposedPlan{
		Mentor: MentorPlan{
			PrimaryID:   "musk",
			SecondaryID: "inamori",
			BlendWeight: 0.7,
			Reasoning:   "Innovation-driven with philosophical depth",
		},
		Board: []SeatPlan{
			{SeatType: "ceo", PersonaID: "musk", Reasoning: "visionary tech leadership"},
			{SeatType: "cto", PersonaID: "grove", Reasoning: "engineering management expertise"},
		},
		OrgDesign: OrgDesignPlan{
			Units: []OrgUnitPlan{
				{RefID: "ceo", ParentRefID: "", Name: "CEO Office", UnitType: "department", HeadRole: "CEO"},
				{RefID: "eng", ParentRefID: "ceo", Name: "Engineering", UnitType: "department", HeadRole: "VP Engineering"},
				{RefID: "sales", ParentRefID: "ceo", Name: "Sales", UnitType: "department", HeadRole: "VP Sales"},
			},
			Reasoning: "Lean org for Series A SaaS",
		},
		Policies: PolicyPlan{
			Framework:        "okr",
			CheckinQuestions: []string{"What did you accomplish?", "Any blockers?", "Plan for tomorrow?"},
			TrackingFocus:    []string{"velocity", "quality"},
			RiskRules: RiskRules{
				ConsecutiveMisses:      3,
				SentimentDropThreshold: -0.3,
				UrgentKeywords:         []string{"urgent", "blocked"},
			},
			Cadence: Cadence{
				DailyActions:   []string{"checkin"},
				WeeklyActions:  []string{"review"},
				WeeklyDay:      "friday",
				MonthlyActions: []string{"retro"},
				MonthlyDay:     1,
			},
			Reasoning: "OKR for fast-moving SaaS",
		},
		Schedule: SchedulePlan{
			Checkin:    "0 9 * * 1-5",
			Chase:      "30 17 * * 1-5",
			Summary:    "0 19 * * 1-5",
			Briefing:   "0 8 * * 1-5",
			SignalScan: "*/30 9-18 * * 1-5",
			Timezone:   "Asia/Manila",
		},
		Reasoning: "Comprehensive plan for B2B SaaS startup",
	}
}

func integrationPlanJSON() string {
	data, _ := json.Marshal(integrationPlan())
	return string(data)
}

// ---------------------------------------------------------------------------
// TestIntegration_FullOnboardingFlow
// ---------------------------------------------------------------------------
// Tests the entire onboarding flow end-to-end:
//  1. Boss sends "/start" -> session created, gets greeting
//  2. Boss sends rich company info -> fields extracted, all required covered
//     -> transitions to configuring -> plan generated -> returns Step 1
//  3. Boss confirms "ok" -> Step 1 applied, returns Step 2
//  4. Boss confirms "ok" -> Step 2 applied, returns Step 3
//  5. Boss confirms "ok" -> Step 3 applied, returns Step 4
//  6. Boss confirms "ok" -> Step 4 applied, onboarding complete

func TestIntegration_FullOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true}

	db := &integrationMockDB{}
	rds := newIntegrationRedis()

	// extractLLM: Call 1 from /start (nothing), Call 2 from rich message (all fields).
	extractLLM := &integrationMockLLM{
		responses: []string{
			`{}`,
			`{"industry":"SaaS","company_stage":"Series A","business_model":"B2B","team_size":15,"org_structure":"3 teams: eng, product, sales","current_projects":"API platform v2","pain_points":["hiring","communication"],"comm_tools":["slack","github"]}`,
		},
	}

	// planLLM: Called once when transitioning to configuring.
	planLLM := &integrationMockLLM{
		responses: []string{integrationPlanJSON()},
	}

	// confirmLLM: Not called in the happy-path (all confirmations are "ok").
	confirmLLM := &integrationMockLLM{}

	// chatLLM: Called once during /start to generate the greeting.
	chatLLM := &integrationMockChatLLM{
		responses: []string{
			"Welcome! I'm your AI management consultant. Tell me about your company.",
		},
	}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// -----------------------------------------------------------------------
	// Step 1: Boss sends "/start" -> gets consultant greeting
	// -----------------------------------------------------------------------
	resp1, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123", "/start")
	if err != nil {
		t.Fatalf("step 1 (/start): unexpected error: %v", err)
	}
	if !strings.Contains(resp1, "Welcome") {
		t.Errorf("step 1: expected greeting with 'Welcome', got: %q", resp1)
	}

	// Verify session was created in onboarding state.
	if !db.sessionExists {
		t.Fatal("step 1: session should exist after /start")
	}
	if db.session.Status != "onboarding" {
		t.Errorf("step 1: expected status 'onboarding', got %q", db.session.Status)
	}
	if db.session.MessageCount != 1 {
		t.Errorf("step 1: expected message count 1, got %d", db.session.MessageCount)
	}

	// -----------------------------------------------------------------------
	// Step 2: Rich message -> all fields filled -> plan generated -> Step 1
	// -----------------------------------------------------------------------
	resp2, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123",
		"We're a SaaS B2B startup at Series A, 15 people in 3 teams (eng, product, sales). "+
			"Working on API platform v2. Main pain points are hiring and communication. "+
			"We use Slack and GitHub.")
	if err != nil {
		t.Fatalf("step 2 (rich message): unexpected error: %v", err)
	}
	if !strings.Contains(resp2, "Step 1") {
		t.Errorf("step 2: expected 'Step 1' in response, got: %q", resp2)
	}
	if !strings.Contains(resp2, "Mentor") {
		t.Errorf("step 2: expected 'Mentor' in response, got: %q", resp2)
	}
	if !strings.Contains(resp2, "musk") {
		t.Errorf("step 2: expected 'musk' (primary mentor) in response, got: %q", resp2)
	}
	if !strings.Contains(resp2, "Reply OK to confirm") {
		t.Errorf("step 2: expected confirmation prompt, got: %q", resp2)
	}

	// Verify session transitioned to confirming step 1.
	if db.session.Status != "confirming" {
		t.Errorf("step 2: expected status 'confirming', got %q", db.session.Status)
	}
	if db.session.ConfirmStep != 1 {
		t.Errorf("step 2: expected confirm step 1, got %d", db.session.ConfirmStep)
	}

	// Verify plan was stored.
	if len(db.session.ProposedPlan) == 0 {
		t.Fatal("step 2: proposed plan should be stored in session")
	}
	var storedPlan ProposedPlan
	if err := json.Unmarshal(db.session.ProposedPlan, &storedPlan); err != nil {
		t.Fatalf("step 2: failed to unmarshal stored plan: %v", err)
	}
	if storedPlan.Mentor.PrimaryID != "musk" {
		t.Errorf("step 2: expected stored mentor 'musk', got %q", storedPlan.Mentor.PrimaryID)
	}

	// -----------------------------------------------------------------------
	// Step 3: Confirm Step 1 ("ok") -> applied, returns Step 2
	// -----------------------------------------------------------------------
	resp3, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123", "ok")
	if err != nil {
		t.Fatalf("step 3 (confirm step 1): unexpected error: %v", err)
	}
	if !strings.Contains(resp3, "Step 2") {
		t.Errorf("step 3: expected 'Step 2' in response, got: %q", resp3)
	}
	if !strings.Contains(resp3, "Organization") {
		t.Errorf("step 3: expected 'Organization' in response, got: %q", resp3)
	}
	if !strings.Contains(resp3, "CEO Office") {
		t.Errorf("step 3: expected 'CEO Office' in response, got: %q", resp3)
	}
	if !strings.Contains(resp3, "Reply OK to confirm") {
		t.Errorf("step 3: expected confirmation prompt, got: %q", resp3)
	}

	// Verify step 1 was applied (UpdateOrg for mentor/board).
	if db.updateOrgCalls < 1 {
		t.Errorf("step 3: expected at least 1 UpdateOrg call for step 1, got %d", db.updateOrgCalls)
	}
	// Verify session advanced to step 2.
	if db.session.ConfirmStep != 2 {
		t.Errorf("step 3: expected confirm step 2, got %d", db.session.ConfirmStep)
	}

	// -----------------------------------------------------------------------
	// Step 4: Confirm Step 2 ("ok") -> applied, returns Step 3
	// -----------------------------------------------------------------------
	resp4, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123", "ok")
	if err != nil {
		t.Fatalf("step 4 (confirm step 2): unexpected error: %v", err)
	}
	if !strings.Contains(resp4, "Step 3") {
		t.Errorf("step 4: expected 'Step 3' in response, got: %q", resp4)
	}
	if !strings.Contains(resp4, "Policies") {
		t.Errorf("step 4: expected 'Policies' in response, got: %q", resp4)
	}
	if !strings.Contains(resp4, "okr") {
		t.Errorf("step 4: expected 'okr' (framework) in response, got: %q", resp4)
	}

	// Verify step 2 was applied (org structure: delete + create).
	if !db.orgUnitsDeleted {
		t.Error("step 4: expected org units to be deleted for step 2")
	}
	if len(db.orgUnitsCreated) != 3 {
		t.Errorf("step 4: expected 3 org units created, got %d: %v", len(db.orgUnitsCreated), db.orgUnitsCreated)
	}
	// Verify session advanced to step 3.
	if db.session.ConfirmStep != 3 {
		t.Errorf("step 4: expected confirm step 3, got %d", db.session.ConfirmStep)
	}

	// -----------------------------------------------------------------------
	// Step 5: Confirm Step 3 ("ok") -> applied, returns Step 4
	// -----------------------------------------------------------------------
	resp5, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123", "ok")
	if err != nil {
		t.Fatalf("step 5 (confirm step 3): unexpected error: %v", err)
	}
	if !strings.Contains(resp5, "Step 4") {
		t.Errorf("step 5: expected 'Step 4' in response, got: %q", resp5)
	}
	if !strings.Contains(resp5, "Schedule") {
		t.Errorf("step 5: expected 'Schedule' in response, got: %q", resp5)
	}
	if !strings.Contains(resp5, "Asia/Manila") {
		t.Errorf("step 5: expected 'Asia/Manila' in response, got: %q", resp5)
	}

	// Verify step 3 was applied (policies -> UpdateOrg).
	if db.updateOrgCalls < 2 {
		t.Errorf("step 5: expected at least 2 UpdateOrg calls (step 1 + step 3), got %d", db.updateOrgCalls)
	}
	// Verify session advanced to step 4.
	if db.session.ConfirmStep != 4 {
		t.Errorf("step 5: expected confirm step 4, got %d", db.session.ConfirmStep)
	}

	// -----------------------------------------------------------------------
	// Step 6: Confirm Step 4 ("ok") -> applied, onboarding complete
	// -----------------------------------------------------------------------
	resp6, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss123", "ok")
	if err != nil {
		t.Fatalf("step 6 (confirm step 4): unexpected error: %v", err)
	}
	if !strings.Contains(strings.ToLower(resp6), "complete") {
		t.Errorf("step 6: expected 'complete' in response, got: %q", resp6)
	}
	if !strings.Contains(resp6, "configured") {
		t.Errorf("step 6: expected 'configured' in response, got: %q", resp6)
	}

	// Verify tenant was marked as completed.
	if !db.tenantCompleted {
		t.Error("step 6: expected SetTenantOnboardingCompleted to be called")
	}

	// Verify final session state is active.
	if db.session.Status != "active" {
		t.Errorf("step 6: expected final status 'active', got %q", db.session.Status)
	}

	// Verify step 4 was applied (schedule -> UpdateOrg).
	// Total: step 1 (mentor/board) + step 3 (policies) + step 4 (schedule) + completion = at least 4.
	if db.updateOrgCalls < 3 {
		t.Errorf("step 6: expected at least 3 UpdateOrg calls, got %d", db.updateOrgCalls)
	}

	// Verify chat history was cleaned up from Redis.
	hKey := historyKey(tenantID)
	rds.mu.Lock()
	_, historyExists := rds.data[hKey]
	rds.mu.Unlock()
	if historyExists {
		t.Error("step 6: chat history should be deleted from Redis after completion")
	}

	// Verify LLM call counts.
	if extractLLM.callCount != 2 {
		t.Errorf("extractLLM: expected 2 calls, got %d", extractLLM.callCount)
	}
	if planLLM.callCount != 1 {
		t.Errorf("planLLM: expected 1 call, got %d", planLLM.callCount)
	}
	if confirmLLM.callCount != 0 {
		t.Errorf("confirmLLM: expected 0 calls in happy path, got %d", confirmLLM.callCount)
	}
	if chatLLM.callCount != 1 {
		t.Errorf("chatLLM: expected 1 call (for /start greeting), got %d", chatLLM.callCount)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_OnboardingWithMultipleMessages
// ---------------------------------------------------------------------------
// Tests that partial data accumulates across multiple messages before
// triggering plan generation.

func TestIntegration_OnboardingWithMultipleMessages(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{2, 2, 2, 2}, Valid: true}

	db := &integrationMockDB{}
	rds := newIntegrationRedis()

	// extractLLM: 3 calls, each returning partial data.
	extractLLM := &integrationMockLLM{
		responses: []string{
			`{}`,
			`{"industry":"FinTech","company_stage":"Seed","business_model":"B2C"}`,
			`{"team_size":8,"org_structure":"2 teams","current_projects":"Mobile app","pain_points":["scaling"],"comm_tools":["discord"]}`,
		},
	}

	planLLM := &integrationMockLLM{responses: []string{integrationPlanJSON()}}
	confirmLLM := &integrationMockLLM{}
	chatLLM := &integrationMockChatLLM{
		responses: []string{
			"Welcome! Tell me about your company.",
			"Great, you're in FinTech! How big is your team and what tools do you use?",
		},
	}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// Message 1: /start
	resp1, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "/start")
	if err != nil {
		t.Fatalf("message 1: unexpected error: %v", err)
	}
	if !strings.Contains(resp1, "Welcome") {
		t.Errorf("message 1: expected 'Welcome', got: %q", resp1)
	}
	if db.session.Status != "onboarding" {
		t.Errorf("message 1: expected status 'onboarding', got %q", db.session.Status)
	}

	// Message 2: partial info (industry, stage, model only).
	resp2, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "We're a FinTech seed stage B2C startup")
	if err != nil {
		t.Fatalf("message 2: unexpected error: %v", err)
	}
	if db.session.Status != "onboarding" {
		t.Errorf("message 2: should still be 'onboarding' after partial info, got %q", db.session.Status)
	}
	if !strings.Contains(resp2, "FinTech") {
		t.Errorf("message 2: expected 'FinTech' in response, got: %q", resp2)
	}

	// Verify collected data has partial fields.
	var collected CollectedData
	if err := json.Unmarshal(db.session.CollectedData, &collected); err != nil {
		t.Fatalf("message 2: failed to unmarshal collected data: %v", err)
	}
	if collected.Industry != "FinTech" {
		t.Errorf("message 2: expected industry 'FinTech', got %q", collected.Industry)
	}
	if collected.CompanyStage != "Seed" {
		t.Errorf("message 2: expected stage 'Seed', got %q", collected.CompanyStage)
	}
	if collected.BusinessModel != "B2C" {
		t.Errorf("message 2: expected model 'B2C', got %q", collected.BusinessModel)
	}
	if collected.TeamSize != 0 {
		t.Errorf("message 2: expected team_size 0 (not yet filled), got %d", collected.TeamSize)
	}

	// Message 3: remaining fields -> triggers plan generation.
	resp3, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1",
		"We have 8 people in 2 teams, building a mobile app. Main issue is scaling. We use Discord.")
	if err != nil {
		t.Fatalf("message 3: unexpected error: %v", err)
	}
	if !strings.Contains(resp3, "Step 1") {
		t.Errorf("message 3: expected 'Step 1' in response (plan review), got: %q", resp3)
	}
	if db.session.Status != "confirming" {
		t.Errorf("message 3: expected status 'confirming', got %q", db.session.Status)
	}

	// Verify chatLLM was called twice (messages 1 and 2, not 3 which triggered plan).
	if chatLLM.callCount != 2 {
		t.Errorf("chatLLM: expected 2 calls, got %d", chatLLM.callCount)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_ConfirmWithModification
// ---------------------------------------------------------------------------
// Tests that the boss can request a modification during confirmation, then confirm.

func TestIntegration_ConfirmWithModification(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{3, 3, 3, 3}, Valid: true}

	// Pre-populate a session already in confirming step 1.
	planJSON, _ := json.Marshal(integrationPlan())
	collectedJSON, _ := json.Marshal(fullCollectedData())

	db := &integrationMockDB{
		session: &sqlc.OnboardingSession{
			ID:            pgtype.UUID{Bytes: [16]byte{0xBB}, Valid: true},
			TenantID:      tenantID,
			Status:        "confirming",
			ConfirmStep:   1,
			CollectedData: collectedJSON,
			ProposedPlan:  planJSON,
			MessageCount:  5,
			ChannelType:   "telegram",
		},
		sessionExists: true,
	}
	rds := newIntegrationRedis()

	// Build modified plan with mentor changed to inamori.
	modifiedPlan := integrationPlan()
	modifiedPlan.Mentor.PrimaryID = "inamori"
	modifiedPlan.Mentor.Reasoning = "Changed per boss request"
	modifiedPlanJSON, _ := json.Marshal(modifiedPlan)

	extractLLM := &integrationMockLLM{}
	planLLM := &integrationMockLLM{}
	confirmLLM := &integrationMockLLM{responses: []string{string(modifiedPlanJSON)}}
	chatLLM := &integrationMockChatLLM{}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// Boss requests a modification.
	resp1, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "change mentor to inamori instead")
	if err != nil {
		t.Fatalf("modification request: unexpected error: %v", err)
	}
	if !strings.Contains(resp1, "inamori") {
		t.Errorf("modification: expected 'inamori' in response, got: %q", resp1)
	}
	if !strings.Contains(resp1, "Step 1") {
		t.Errorf("modification: expected 'Step 1' in response (still on step 1), got: %q", resp1)
	}

	// Verify plan was updated in session.
	var updatedPlan ProposedPlan
	if err := json.Unmarshal(db.session.ProposedPlan, &updatedPlan); err != nil {
		t.Fatalf("failed to unmarshal updated plan: %v", err)
	}
	if updatedPlan.Mentor.PrimaryID != "inamori" {
		t.Errorf("stored plan should reflect modification: expected mentor 'inamori', got %q", updatedPlan.Mentor.PrimaryID)
	}

	// Boss now confirms.
	resp2, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "ok")
	if err != nil {
		t.Fatalf("confirm after modification: unexpected error: %v", err)
	}
	if !strings.Contains(resp2, "Step 2") {
		t.Errorf("confirm: expected 'Step 2' in response, got: %q", resp2)
	}
	if db.session.ConfirmStep != 2 {
		t.Errorf("confirm: expected confirm step 2, got %d", db.session.ConfirmStep)
	}

	// Verify confirmLLM was called exactly once (for modification).
	if confirmLLM.callCount != 1 {
		t.Errorf("confirmLLM: expected 1 call, got %d", confirmLLM.callCount)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_ConcurrentMessagesAreLocked
// ---------------------------------------------------------------------------
// Tests that a second message while a lock is held gets the lock-wait response.

func TestIntegration_ConcurrentMessagesAreLocked(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{4, 4, 4, 4}, Valid: true}

	db := &integrationMockDB{
		session: &sqlc.OnboardingSession{
			ID:            pgtype.UUID{Bytes: [16]byte{0xCC}, Valid: true},
			TenantID:      tenantID,
			Status:        "onboarding",
			CollectedData: []byte("{}"),
			ChannelType:   "telegram",
		},
		sessionExists: true,
	}
	rds := newIntegrationRedis()

	extractLLM := &integrationMockLLM{responses: []string{`{}`}}
	planLLM := &integrationMockLLM{}
	confirmLLM := &integrationMockLLM{}
	chatLLM := &integrationMockChatLLM{responses: []string{"Tell me more."}}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// First message processes normally (acquires lock, then releases on defer).
	resp1, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "Hello")
	if err != nil {
		t.Fatalf("first message: unexpected error: %v", err)
	}
	if resp1 != "Tell me more." {
		t.Errorf("first message: expected 'Tell me more.', got: %q", resp1)
	}

	// Pre-set the lock key to simulate a concurrent request holding the lock.
	// The lock key format is "onboarding:lock:%x" where %x is the UUID bytes in hex.
	lockKey := fmt.Sprintf("onboarding:lock:%x", tenantID.Bytes)
	rds.mu.Lock()
	rds.data[lockKey] = "1"
	rds.mu.Unlock()

	// Second message should see the lock and return a "thinking" response.
	resp2, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "Another message")
	if err != nil {
		t.Fatalf("locked message: unexpected error: %v", err)
	}
	if !strings.Contains(resp2, "still thinking") {
		t.Errorf("locked message: expected 'still thinking' response, got: %q", resp2)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_AlreadyCompletedSessionRejects
// ---------------------------------------------------------------------------

func TestIntegration_AlreadyCompletedSessionRejects(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{5, 5, 5, 5}, Valid: true}

	db := &integrationMockDB{
		session: &sqlc.OnboardingSession{
			TenantID: tenantID,
			Status:   "active",
		},
		sessionExists: true,
	}
	rds := newIntegrationRedis()
	svc := NewService(db, rds, &integrationMockLLM{}, &integrationMockLLM{}, &integrationMockLLM{}, &integrationMockChatLLM{})

	resp, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp, "already complete") {
		t.Errorf("expected 'already complete' message, got: %q", resp)
	}
}

// ---------------------------------------------------------------------------
// TestIntegration_ChatHistoryAccumulates
// ---------------------------------------------------------------------------
// Tests that chat history is accumulated across multiple onboarding messages.

func TestIntegration_ChatHistoryAccumulates(t *testing.T) {
	ctx := context.Background()
	tenantID := pgtype.UUID{Bytes: [16]byte{6, 6, 6, 6}, Valid: true}

	db := &integrationMockDB{}
	rds := newIntegrationRedis()

	extractLLM := &integrationMockLLM{responses: []string{`{}`, `{}`, `{}`}}
	planLLM := &integrationMockLLM{}
	confirmLLM := &integrationMockLLM{}
	chatLLM := &integrationMockChatLLM{
		responses: []string{
			"Welcome! What industry?",
			"How big is your team?",
			"What tools do you use?",
		},
	}

	svc := NewService(db, rds, extractLLM, planLLM, confirmLLM, chatLLM)

	// Send 3 messages.
	msgs := []string{"hi", "SaaS company", "15 people"}
	for i, msg := range msgs {
		if _, err := svc.HandleMessage(ctx, tenantID, "telegram", "boss1", msg); err != nil {
			t.Fatalf("message %d (%q): unexpected error: %v", i+1, msg, err)
		}
	}

	// Verify history has 6 messages (3 user + 3 assistant).
	hKey := historyKey(tenantID)
	rds.mu.Lock()
	historyData, exists := rds.data[hKey]
	rds.mu.Unlock()
	if !exists {
		t.Fatal("chat history should exist in Redis after 3 messages")
	}

	var history []brain.ChatMessage
	if err := json.Unmarshal([]byte(historyData), &history); err != nil {
		t.Fatalf("failed to unmarshal chat history: %v", err)
	}
	if len(history) != 6 {
		t.Fatalf("expected 6 messages in history (3 user + 3 assistant), got %d", len(history))
	}

	// Verify message pairs are in correct order.
	expectedPairs := []struct {
		role    string
		content string
	}{
		{"user", "hi"},
		{"assistant", "Welcome! What industry?"},
		{"user", "SaaS company"},
		{"assistant", "How big is your team?"},
		{"user", "15 people"},
		{"assistant", "What tools do you use?"},
	}

	for i, exp := range expectedPairs {
		if history[i].Role != exp.role {
			t.Errorf("history[%d]: expected role %q, got %q", i, exp.role, history[i].Role)
		}
		if history[i].Content != exp.content {
			t.Errorf("history[%d]: expected content %q, got %q", i, exp.content, history[i].Content)
		}
	}
}
