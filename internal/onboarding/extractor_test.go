package onboarding

import (
	"context"
	"errors"
	"testing"
)

// mockLLM implements brain.LLMClient for testing.
type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Chat(_ context.Context, _, _ string) (string, error) {
	return m.response, m.err
}

func TestExtractInfo_NewInfo(t *testing.T) {
	mock := &mockLLM{response: `{"industry":"SaaS","team_size":15}`}
	ext := NewExtractor(mock)
	current := &CollectedData{}

	result, err := ext.ExtractInfo(context.Background(), current, "We are a SaaS company with 15 people")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Industry != "SaaS" {
		t.Errorf("expected industry SaaS, got %q", result.Industry)
	}
	if result.TeamSize != 15 {
		t.Errorf("expected team_size 15, got %d", result.TeamSize)
	}
	// Original must be unchanged (immutability).
	if current.Industry != "" {
		t.Error("original data was mutated")
	}
}

func TestExtractInfo_NoNewInfo(t *testing.T) {
	mock := &mockLLM{response: `{}`}
	ext := NewExtractor(mock)
	current := &CollectedData{Industry: "FinTech", TeamSize: 10}

	result, err := ext.ExtractInfo(context.Background(), current, "nothing useful here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Industry != "FinTech" {
		t.Errorf("expected industry FinTech, got %q", result.Industry)
	}
	if result.TeamSize != 10 {
		t.Errorf("expected team_size 10, got %d", result.TeamSize)
	}
}

func TestExtractInfo_LLMError(t *testing.T) {
	mock := &mockLLM{err: errors.New("API timeout")}
	ext := NewExtractor(mock)
	current := &CollectedData{Industry: "HealthTech", TeamSize: 5}

	result, err := ext.ExtractInfo(context.Background(), current, "some message")
	if err != nil {
		t.Fatalf("expected nil error on LLM failure, got: %v", err)
	}

	// Data should be unchanged.
	if result.Industry != "HealthTech" {
		t.Errorf("expected industry HealthTech, got %q", result.Industry)
	}
	if result.TeamSize != 5 {
		t.Errorf("expected team_size 5, got %d", result.TeamSize)
	}
}

func TestExtractInfo_InvalidJSON(t *testing.T) {
	mock := &mockLLM{response: `not valid json at all`}
	ext := NewExtractor(mock)
	current := &CollectedData{Industry: "EdTech", TeamSize: 20}

	result, err := ext.ExtractInfo(context.Background(), current, "blah blah")
	if err != nil {
		t.Fatalf("expected nil error on invalid JSON, got: %v", err)
	}

	// Data should be unchanged.
	if result.Industry != "EdTech" {
		t.Errorf("expected industry EdTech, got %q", result.Industry)
	}
	if result.TeamSize != 20 {
		t.Errorf("expected team_size 20, got %d", result.TeamSize)
	}
}

func TestExtractInfo_MarkdownWrappedJSON(t *testing.T) {
	mock := &mockLLM{response: "```json\n{\"industry\":\"Retail\"}\n```"}
	ext := NewExtractor(mock)
	current := &CollectedData{}

	result, err := ext.ExtractInfo(context.Background(), current, "We sell retail goods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Industry != "Retail" {
		t.Errorf("expected industry Retail, got %q", result.Industry)
	}
}

func TestMergeCollectedData_PreservesExisting(t *testing.T) {
	base := &CollectedData{
		Industry:        "SaaS",
		CompanyStage:    "Growth",
		BusinessModel:   "B2B",
		TeamSize:        25,
		OrgStructure:    "Flat",
		CurrentProjects: "Platform v2",
		PainPoints:      []string{"hiring"},
		CommTools:       []string{"Slack"},
		CulturePrefs:    "Collaborative",
		GoalFramework:   "OKR",
	}

	// Delta only updates team_size and adds pain_points.
	delta := &CollectedData{
		TeamSize:   30,
		PainPoints: []string{"hiring", "retention"},
	}

	result := mergeCollectedData(base, delta)

	// Updated fields.
	if result.TeamSize != 30 {
		t.Errorf("expected team_size 30, got %d", result.TeamSize)
	}
	if len(result.PainPoints) != 2 || result.PainPoints[1] != "retention" {
		t.Errorf("expected pain_points [hiring, retention], got %v", result.PainPoints)
	}

	// Preserved fields.
	if result.Industry != "SaaS" {
		t.Errorf("expected industry SaaS, got %q", result.Industry)
	}
	if result.CompanyStage != "Growth" {
		t.Errorf("expected company_stage Growth, got %q", result.CompanyStage)
	}
	if result.BusinessModel != "B2B" {
		t.Errorf("expected business_model B2B, got %q", result.BusinessModel)
	}
	if result.OrgStructure != "Flat" {
		t.Errorf("expected org_structure Flat, got %q", result.OrgStructure)
	}
	if result.CurrentProjects != "Platform v2" {
		t.Errorf("expected current_projects Platform v2, got %q", result.CurrentProjects)
	}
	if len(result.CommTools) != 1 || result.CommTools[0] != "Slack" {
		t.Errorf("expected comm_tools [Slack], got %v", result.CommTools)
	}
	if result.CulturePrefs != "Collaborative" {
		t.Errorf("expected culture_prefs Collaborative, got %q", result.CulturePrefs)
	}
	if result.GoalFramework != "OKR" {
		t.Errorf("expected goal_framework OKR, got %q", result.GoalFramework)
	}

	// Base must be unchanged (immutability).
	if base.TeamSize != 25 {
		t.Error("base data was mutated (team_size)")
	}
	if len(base.PainPoints) != 1 {
		t.Error("base data was mutated (pain_points)")
	}
}

func TestMergeCollectedData_EmptyDelta(t *testing.T) {
	base := &CollectedData{
		Industry:   "SaaS",
		TeamSize:   10,
		PainPoints: []string{"scaling"},
	}
	delta := &CollectedData{}

	result := mergeCollectedData(base, delta)

	if result.Industry != "SaaS" {
		t.Errorf("expected industry SaaS, got %q", result.Industry)
	}
	if result.TeamSize != 10 {
		t.Errorf("expected team_size 10, got %d", result.TeamSize)
	}
	if len(result.PainPoints) != 1 || result.PainPoints[0] != "scaling" {
		t.Errorf("expected pain_points [scaling], got %v", result.PainPoints)
	}
}

func TestMergeCollectedData_EmptyBase(t *testing.T) {
	base := &CollectedData{}
	delta := &CollectedData{
		Industry:     "Gaming",
		TeamSize:     50,
		CulturePrefs: "Fast-paced",
	}

	result := mergeCollectedData(base, delta)

	if result.Industry != "Gaming" {
		t.Errorf("expected industry Gaming, got %q", result.Industry)
	}
	if result.TeamSize != 50 {
		t.Errorf("expected team_size 50, got %d", result.TeamSize)
	}
	if result.CulturePrefs != "Fast-paced" {
		t.Errorf("expected culture_prefs Fast-paced, got %q", result.CulturePrefs)
	}
}
