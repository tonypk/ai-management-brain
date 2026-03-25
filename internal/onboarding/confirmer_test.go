package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockConfirmerLLM is a test double for brain.LLMClient, specific to confirmer tests.
// Using a different name to avoid redeclaration with mockLLM in extractor_test.go.
type mockConfirmerLLM struct {
	response string
	err      error
}

func (m *mockConfirmerLLM) Chat(_ context.Context, _, _ string) (string, error) {
	return m.response, m.err
}

func samplePlan() *ProposedPlan {
	return &ProposedPlan{
		Mentor: MentorPlan{
			PrimaryID:   "musk",
			SecondaryID: "inamori",
			BlendWeight: 0.7,
			Reasoning:   "Innovation + philosophy blend",
		},
		Board: []SeatPlan{
			{SeatType: "ceo", PersonaID: "musk", Reasoning: "visionary leadership"},
			{SeatType: "cto", PersonaID: "grove", Reasoning: "engineering excellence"},
			{SeatType: "chro", PersonaID: "inamori", Reasoning: "people-first culture"},
		},
		OrgDesign: OrgDesignPlan{
			Units: []OrgUnitPlan{
				{RefID: "ceo", ParentRefID: "", Name: "CEO Office", UnitType: "department", HeadRole: "CEO"},
				{RefID: "eng", ParentRefID: "ceo", Name: "Engineering", UnitType: "department", HeadRole: "VP Engineering"},
				{RefID: "fe", ParentRefID: "eng", Name: "Frontend Team", UnitType: "team", HeadRole: "Team Lead"},
				{RefID: "be", ParentRefID: "eng", Name: "Backend Team", UnitType: "team", HeadRole: "Team Lead"},
				{RefID: "prod", ParentRefID: "ceo", Name: "Product", UnitType: "department", HeadRole: "Product Manager"},
			},
			Reasoning: "Standard tech org structure",
		},
		Policies: PolicyPlan{
			Framework:        "okr",
			CheckinQuestions: []string{"What did you accomplish today?", "Any blockers?", "What's your plan for tomorrow?"},
			TrackingFocus:    []string{"velocity", "quality"},
			RiskRules: RiskRules{
				ConsecutiveMisses:      3,
				SentimentDropThreshold: -0.3,
				UrgentKeywords:         []string{"urgent", "blocked", "help"},
			},
			Cadence: Cadence{
				DailyActions:   []string{"checkin", "standup"},
				WeeklyActions:  []string{"review", "retro"},
				WeeklyDay:      "friday",
				MonthlyActions: []string{"all-hands", "performance review"},
				MonthlyDay:     1,
			},
			Reasoning: "OKR framework suits fast-moving SaaS",
		},
		Schedule: SchedulePlan{
			Checkin:    "0 9 * * 1-5",
			Chase:      "30 17 * * 1-5",
			Summary:    "0 19 * * 1-5",
			Briefing:   "0 8 * * 1-5",
			SignalScan: "*/30 9-18 * * 1-5",
			Timezone:   "Asia/Manila",
		},
		Reasoning: "Comprehensive plan for SaaS team",
	}
}

func TestFormatStep_MentorAndBoard(t *testing.T) {
	c := NewConfirmer(nil)
	plan := samplePlan()

	result := c.FormatStep(plan, 1)

	// Must contain mentor name.
	if !strings.Contains(result, "musk") {
		t.Error("expected step 1 to contain primary mentor 'musk'")
	}
	// Must contain secondary mentor.
	if !strings.Contains(result, "inamori") {
		t.Error("expected step 1 to contain secondary mentor 'inamori'")
	}
	// Must contain board seats.
	if !strings.Contains(result, "CEO") {
		t.Error("expected step 1 to contain board seat 'CEO'")
	}
	if !strings.Contains(result, "CTO") {
		t.Error("expected step 1 to contain board seat 'CTO'")
	}
	if !strings.Contains(result, "CHRO") {
		t.Error("expected step 1 to contain board seat 'CHRO'")
	}
	// Must contain board seat personas.
	if !strings.Contains(result, "grove") {
		t.Error("expected step 1 to contain board persona 'grove'")
	}
	// Must contain confirmation prompt.
	if !strings.Contains(result, "Reply OK to confirm") {
		t.Error("expected step 1 to end with confirmation prompt")
	}
}

func TestFormatStep_OrgStructure(t *testing.T) {
	c := NewConfirmer(nil)
	plan := samplePlan()

	result := c.FormatStep(plan, 2)

	// Must contain org unit names in tree form.
	if !strings.Contains(result, "CEO Office") {
		t.Error("expected step 2 to contain 'CEO Office'")
	}
	if !strings.Contains(result, "Engineering") {
		t.Error("expected step 2 to contain 'Engineering'")
	}
	if !strings.Contains(result, "Frontend Team") {
		t.Error("expected step 2 to contain 'Frontend Team'")
	}
	if !strings.Contains(result, "Backend Team") {
		t.Error("expected step 2 to contain 'Backend Team'")
	}
	if !strings.Contains(result, "Product") {
		t.Error("expected step 2 to contain 'Product'")
	}
	// Must contain tree connectors.
	if !strings.Contains(result, "\u251c") && !strings.Contains(result, "\u2514") {
		t.Error("expected step 2 to contain tree connectors")
	}
	// Must contain head roles.
	if !strings.Contains(result, "VP Engineering") {
		t.Error("expected step 2 to contain 'VP Engineering'")
	}
	if !strings.Contains(result, "Team Lead") {
		t.Error("expected step 2 to contain 'Team Lead'")
	}
	// Must contain confirmation prompt.
	if !strings.Contains(result, "Reply OK to confirm") {
		t.Error("expected step 2 to end with confirmation prompt")
	}
}

func TestFormatStep_Policies(t *testing.T) {
	c := NewConfirmer(nil)
	plan := samplePlan()

	result := c.FormatStep(plan, 3)

	// Must contain framework.
	if !strings.Contains(result, "okr") {
		t.Error("expected step 3 to contain framework 'okr'")
	}
	// Must contain checkin questions.
	if !strings.Contains(result, "What did you accomplish today?") {
		t.Error("expected step 3 to contain checkin question")
	}
	if !strings.Contains(result, "Any blockers?") {
		t.Error("expected step 3 to contain 'Any blockers?'")
	}
	// Must contain risk rules.
	if !strings.Contains(result, "3 days") {
		t.Error("expected step 3 to contain consecutive misses '3 days'")
	}
	// Must contain cadence.
	if !strings.Contains(result, "friday") {
		t.Error("expected step 3 to contain weekly day 'friday'")
	}
	// Must contain confirmation prompt.
	if !strings.Contains(result, "Reply OK to confirm") {
		t.Error("expected step 3 to end with confirmation prompt")
	}
}

func TestFormatStep_Schedule(t *testing.T) {
	c := NewConfirmer(nil)
	plan := samplePlan()

	result := c.FormatStep(plan, 4)

	// Must contain timezone.
	if !strings.Contains(result, "Asia/Manila") {
		t.Error("expected step 4 to contain timezone 'Asia/Manila'")
	}
	// Must contain schedule crons.
	if !strings.Contains(result, "0 9 * * 1-5") {
		t.Error("expected step 4 to contain checkin cron")
	}
	// Must contain human-readable descriptions.
	if !strings.Contains(result, "Mon-Fri") {
		t.Error("expected step 4 to contain 'Mon-Fri'")
	}
	// Must contain schedule labels.
	if !strings.Contains(result, "Check-in") {
		t.Error("expected step 4 to contain 'Check-in'")
	}
	if !strings.Contains(result, "Chase") {
		t.Error("expected step 4 to contain 'Chase'")
	}
	// Must contain confirmation prompt.
	if !strings.Contains(result, "Reply OK to confirm") {
		t.Error("expected step 4 to end with confirmation prompt")
	}
}

func TestIsConfirmation_True(t *testing.T) {
	c := NewConfirmer(nil)

	trueInputs := []string{
		"ok", "OK", "Ok", "  ok  ",
		"yes", "YES", "Yes",
		"confirm", "CONFIRM",
		"好", "好的", "可以", "没问题", "确认",
		"good", "GOOD",
		"looks good", "Looks Good",
		"lgtm", "LGTM",
	}

	for _, input := range trueInputs {
		if !c.IsConfirmation(input) {
			t.Errorf("expected IsConfirmation(%q) to be true", input)
		}
	}
}

func TestIsConfirmation_False(t *testing.T) {
	c := NewConfirmer(nil)

	falseInputs := []string{
		"change mentor to inamori",
		"no",
		"wait",
		"not sure",
		"let me think",
		"modify the schedule",
		"",
	}

	for _, input := range falseInputs {
		if c.IsConfirmation(input) {
			t.Errorf("expected IsConfirmation(%q) to be false", input)
		}
	}
}

func TestHandleModification_Success(t *testing.T) {
	// Return a plan with mentor changed to inamori.
	modifiedPlanJSON := `{
		"mentor": {"primary_id": "inamori", "reasoning": "changed per request"},
		"board": [{"seat_type": "ceo", "persona_id": "inamori", "reasoning": "test"}],
		"org_design": {
			"units": [{"ref_id": "ceo", "name": "CEO Office", "unit_type": "department", "head_role": "CEO"}],
			"reasoning": "test"
		},
		"policies": {
			"framework": "okr",
			"checkin_questions": ["How are things?"],
			"risk_rules": {"consecutive_misses": 3, "sentiment_drop_threshold": -0.3, "urgent_keywords": ["urgent"]},
			"cadence": {"daily_actions": ["checkin"], "weekly_actions": ["review"], "weekly_day": "friday", "monthly_actions": ["retro"], "monthly_day": 1},
			"reasoning": "test"
		},
		"schedule": {"checkin": "0 9 * * 1-5", "timezone": "Asia/Manila"},
		"reasoning": "updated"
	}`

	mock := &mockConfirmerLLM{response: modifiedPlanJSON}
	c := NewConfirmer(mock)
	plan := samplePlan()

	updated, formatted, err := c.HandleModification(context.Background(), plan, 1, "change mentor to inamori")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the plan was updated.
	if updated.Mentor.PrimaryID != "inamori" {
		t.Errorf("expected primary mentor 'inamori', got %q", updated.Mentor.PrimaryID)
	}

	// Verify formatted output contains the updated info.
	if !strings.Contains(formatted, "inamori") {
		t.Error("expected formatted output to contain 'inamori'")
	}

	// Original plan must be unchanged (immutability).
	if plan.Mentor.PrimaryID != "musk" {
		t.Error("original plan was mutated")
	}
}

func TestHandleModification_LLMError(t *testing.T) {
	mock := &mockConfirmerLLM{err: errors.New("API timeout")}
	c := NewConfirmer(mock)
	plan := samplePlan()

	_, _, err := c.HandleModification(context.Background(), plan, 1, "change something")
	if err == nil {
		t.Fatal("expected error on LLM failure")
	}
	if !strings.Contains(err.Error(), "LLM call failed") {
		t.Errorf("expected 'LLM call failed' in error, got: %v", err)
	}
}

func TestHandleModification_InvalidJSON(t *testing.T) {
	mock := &mockConfirmerLLM{response: "this is not json"}
	c := NewConfirmer(mock)
	plan := samplePlan()

	_, _, err := c.HandleModification(context.Background(), plan, 1, "change something")
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected 'failed to parse' in error, got: %v", err)
	}
}

func TestHandleModification_InvalidPlan(t *testing.T) {
	// Valid JSON but missing required fields.
	mock := &mockConfirmerLLM{response: `{"mentor":{"primary_id":""},"board":[],"org_design":{"units":[]},"policies":{"framework":""},"schedule":{"timezone":""}}`}
	c := NewConfirmer(mock)
	plan := samplePlan()

	_, _, err := c.HandleModification(context.Background(), plan, 1, "change something")
	if err == nil {
		t.Fatal("expected error on invalid plan")
	}
	if !strings.Contains(err.Error(), "modified plan is invalid") {
		t.Errorf("expected 'modified plan is invalid' in error, got: %v", err)
	}
}

func TestBuildOrgTree_Empty(t *testing.T) {
	result := buildOrgTree(nil)
	if !strings.Contains(result, "no units defined") {
		t.Errorf("expected 'no units defined' for empty units, got %q", result)
	}
}

func TestBuildOrgTree_SingleRoot(t *testing.T) {
	units := []OrgUnitPlan{
		{RefID: "ceo", ParentRefID: "", Name: "CEO Office", HeadRole: "CEO"},
	}
	result := buildOrgTree(units)
	if !strings.Contains(result, "CEO Office (CEO)") {
		t.Errorf("expected 'CEO Office (CEO)' in tree, got %q", result)
	}
}

func TestBuildOrgTree_NestedStructure(t *testing.T) {
	units := []OrgUnitPlan{
		{RefID: "ceo", ParentRefID: "", Name: "CEO", HeadRole: "You"},
		{RefID: "eng", ParentRefID: "ceo", Name: "Engineering", HeadRole: "VP Eng"},
		{RefID: "fe", ParentRefID: "eng", Name: "Frontend Team", HeadRole: "Team Lead"},
		{RefID: "be", ParentRefID: "eng", Name: "Backend Team", HeadRole: "Team Lead"},
		{RefID: "prod", ParentRefID: "ceo", Name: "Product", HeadRole: "Product Manager"},
	}
	result := buildOrgTree(units)

	// Verify tree structure.
	if !strings.Contains(result, "CEO (You)") {
		t.Error("expected root 'CEO (You)'")
	}
	if !strings.Contains(result, "Engineering (VP Eng)") {
		t.Error("expected 'Engineering (VP Eng)'")
	}
	if !strings.Contains(result, "Frontend Team (Team Lead)") {
		t.Error("expected 'Frontend Team (Team Lead)'")
	}
	if !strings.Contains(result, "Backend Team (Team Lead)") {
		t.Error("expected 'Backend Team (Team Lead)'")
	}
	if !strings.Contains(result, "Product (Product Manager)") {
		t.Error("expected 'Product (Product Manager)'")
	}
	// Verify tree connectors are present.
	if !strings.Contains(result, "\u251c") {
		t.Error("expected branch connector in tree")
	}
	if !strings.Contains(result, "\u2514") {
		t.Error("expected last-branch connector in tree")
	}
}

func TestDescribeCron(t *testing.T) {
	tests := []struct {
		cron     string
		contains string
	}{
		{"0 9 * * 1-5", "Mon-Fri"},
		{"30 17 * * 1-5", "Mon-Fri"},
		{"*/30 9-18 * * 1-5", "every 30 min"},
		{"0 8 * * *", "every day"},
	}

	for _, tc := range tests {
		result := describeCron(tc.cron)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("describeCron(%q) = %q, expected to contain %q", tc.cron, result, tc.contains)
		}
	}
}

func TestNewConfirmer(t *testing.T) {
	mock := &mockConfirmerLLM{}
	c := NewConfirmer(mock)
	if c == nil {
		t.Fatal("expected non-nil confirmer")
	}
	if c.llm != mock {
		t.Error("expected confirmer to hold the provided LLM client")
	}
}

func TestFormatStep_Unknown(t *testing.T) {
	c := NewConfirmer(nil)
	plan := samplePlan()
	result := c.FormatStep(plan, 99)
	if !strings.Contains(result, "Unknown step") {
		t.Errorf("expected 'Unknown step' for invalid step, got %q", result)
	}
}
