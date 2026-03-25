package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockSequenceLLM is a test double for brain.LLMClient that returns a sequence
// of preconfigured responses, advancing on each call.
type mockSequenceLLM struct {
	responses []string
	errs      []error
	callCount int
}

func (m *mockSequenceLLM) Chat(_ context.Context, _, _ string) (string, error) {
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

const validPlanJSON = `{"mentor":{"primary_id":"musk","reasoning":"test"},"board":[{"seat_type":"ceo","persona_id":"musk","reasoning":"test"}],"org_design":{"units":[{"ref_id":"eng","name":"Engineering","unit_type":"department"}],"reasoning":"test"},"policies":{"framework":"okr","checkin_questions":["q1"],"risk_rules":{"consecutive_misses":3,"sentiment_drop_threshold":-0.3,"urgent_keywords":["urgent"]},"cadence":{"daily_actions":["checkin"],"weekly_actions":["review"],"weekly_day":"friday","monthly_actions":["retro"],"monthly_day":1},"reasoning":"test"},"schedule":{"checkin":"0 9 * * 1-5","timezone":"Asia/Manila"},"reasoning":"test"}`

func sampleCollectedData() *CollectedData {
	return &CollectedData{
		Industry:        "SaaS",
		CompanyStage:    "Series A",
		BusinessModel:   "B2B",
		TeamSize:        25,
		OrgStructure:    "Flat",
		CurrentProjects: "Platform v2",
		PainPoints:      []string{"communication", "deadlines"},
		CommTools:       []string{"Slack", "Zoom"},
	}
}

func TestGeneratePlan_ValidOnFirstAttempt(t *testing.T) {
	mock := &mockSequenceLLM{
		responses: []string{validPlanJSON},
		errs:      []error{nil},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Mentor.PrimaryID != "musk" {
		t.Errorf("expected mentor primary_id 'musk', got %q", plan.Mentor.PrimaryID)
	}
	if len(plan.Board) != 1 || plan.Board[0].SeatType != "ceo" {
		t.Error("expected one board seat with seat_type 'ceo'")
	}
	if plan.Policies.Framework != "okr" {
		t.Errorf("expected framework 'okr', got %q", plan.Policies.Framework)
	}
	if plan.Schedule.Timezone != "Asia/Manila" {
		t.Errorf("expected timezone 'Asia/Manila', got %q", plan.Schedule.Timezone)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 LLM call, got %d", mock.callCount)
	}
}

func TestGeneratePlan_InvalidJSONThenValid(t *testing.T) {
	mock := &mockSequenceLLM{
		responses: []string{
			"this is not json at all",
			validPlanJSON,
		},
		errs: []error{nil, nil},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Mentor.PrimaryID != "musk" {
		t.Errorf("expected mentor primary_id 'musk', got %q", plan.Mentor.PrimaryID)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mock.callCount)
	}
}

func TestGeneratePlan_ValidationFailThenValid(t *testing.T) {
	// First response is valid JSON but missing required fields (no mentor primary_id).
	incompletePlan := `{"mentor":{"primary_id":""},"board":[],"org_design":{"units":[]},"policies":{"framework":""},"schedule":{"timezone":""},"reasoning":"incomplete"}`

	mock := &mockSequenceLLM{
		responses: []string{
			incompletePlan,
			validPlanJSON,
		},
		errs: []error{nil, nil},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Mentor.PrimaryID != "musk" {
		t.Errorf("expected mentor primary_id 'musk', got %q", plan.Mentor.PrimaryID)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", mock.callCount)
	}
}

func TestGeneratePlan_AllAttemptsFail(t *testing.T) {
	mock := &mockSequenceLLM{
		responses: []string{
			"bad json 1",
			"bad json 2",
			"bad json 3",
		},
		errs: []error{nil, nil, nil},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	if plan != nil {
		t.Error("expected nil plan on failure")
	}
	if !strings.Contains(err.Error(), "failed to generate valid plan after 3 attempts") {
		t.Errorf("unexpected error message: %v", err)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 LLM calls, got %d", mock.callCount)
	}
}

func TestGeneratePlan_LLMErrorNoRetry(t *testing.T) {
	mock := &mockSequenceLLM{
		responses: []string{},
		errs:      []error{errors.New("network timeout")},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err == nil {
		t.Fatal("expected error on LLM failure")
	}
	if plan != nil {
		t.Error("expected nil plan on LLM error")
	}
	if !strings.Contains(err.Error(), "LLM call failed") {
		t.Errorf("expected 'LLM call failed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("expected wrapped error 'network timeout', got: %v", err)
	}
	// LLM errors should NOT retry — only 1 call.
	if mock.callCount != 1 {
		t.Errorf("expected 1 LLM call (no retry on LLM error), got %d", mock.callCount)
	}
}

func TestGeneratePlan_MarkdownWrappedJSON(t *testing.T) {
	// Verify that cleanJSON strips markdown code fences.
	wrappedJSON := "```json\n" + validPlanJSON + "\n```"

	mock := &mockSequenceLLM{
		responses: []string{wrappedJSON},
		errs:      []error{nil},
	}
	planner := NewPlanner(mock)

	plan, err := planner.GeneratePlan(context.Background(), sampleCollectedData())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Mentor.PrimaryID != "musk" {
		t.Errorf("expected mentor primary_id 'musk', got %q", plan.Mentor.PrimaryID)
	}
}

func TestNewPlanner(t *testing.T) {
	mock := &mockLLM{}
	planner := NewPlanner(mock)
	if planner == nil {
		t.Fatal("expected non-nil planner")
	}
	if planner.llm != mock {
		t.Error("expected planner to hold the provided LLM client")
	}
}

func TestBuildPlanGenerationPrompt(t *testing.T) {
	prompt := buildPlanGenerationPrompt()
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	// Should reference available mentor IDs.
	if !strings.Contains(prompt, "musk") {
		t.Error("expected 'musk' in mentor list")
	}
	if !strings.Contains(prompt, "inamori") {
		t.Error("expected 'inamori' in mentor list")
	}
	// Should instruct JSON-only response.
	if !strings.Contains(prompt, "JSON ONLY") {
		t.Error("expected 'JSON ONLY' instruction")
	}
}
