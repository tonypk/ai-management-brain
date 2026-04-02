package worldmodel

import (
	"encoding/json"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestEvalBlockerEscalation_Triggers(t *testing.T) {
	input := BlockerEscalationInput{
		EmployeeID:      "abc-123",
		EmployeeName:    "Alice",
		Category:        "cross_team",
		Description:     "Waiting for backend API",
		RecurrenceCount: 4,
		FirstSeenAt:     "2026-03-15",
	}
	rec := EvalBlockerEscalation(input)
	if rec == nil {
		t.Fatal("expected recommendation, got nil")
	}
	if rec.Priority != "high" {
		t.Errorf("expected priority high, got %s", rec.Priority)
	}
	if rec.Category != "people" {
		t.Errorf("expected category people, got %s", rec.Category)
	}

	var actions []map[string]any
	if err := json.Unmarshal(rec.SuggestedActions, &actions); err != nil {
		t.Fatalf("failed to unmarshal actions: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(actions))
	}
}

func TestEvalBlockerEscalation_BelowThreshold(t *testing.T) {
	input := BlockerEscalationInput{
		EmployeeID:      "abc-123",
		EmployeeName:    "Alice",
		Category:        "tooling",
		Description:     "IDE crash",
		RecurrenceCount: 2,
	}
	rec := EvalBlockerEscalation(input)
	if rec != nil {
		t.Error("expected nil for recurrence < 3")
	}
}

func TestEvalSkillMatch_Triggers(t *testing.T) {
	input := SkillMatchInput{
		BlockedEmployeeID:   "blocked-id",
		BlockedEmployeeName: "Bob",
		BlockerCategory:     "database",
		HelperEmployeeID:    "helper-id",
		HelperEmployeeName:  "Carol",
		HelperSkillName:     "database_optimization",
		HelperConfidence:    0.85,
	}
	rec := EvalSkillMatch(input)
	if rec == nil {
		t.Fatal("expected recommendation, got nil")
	}
	if rec.Priority != "medium" {
		t.Errorf("expected priority medium, got %s", rec.Priority)
	}

	var actions []map[string]any
	if err := json.Unmarshal(rec.SuggestedActions, &actions); err != nil {
		t.Fatalf("failed to unmarshal actions: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("expected 2 actions (notify both), got %d", len(actions))
	}
}

func TestEvalCompoundRisk_Triggers(t *testing.T) {
	input := CompoundRiskInput{
		EmployeeID:     "emp-id",
		EmployeeName:   "Dave",
		SentimentTrend: []string{"negative", "negative", "neutral"},
		ActiveBlockers: []string{"cross_team", "tooling"},
	}
	rec := EvalCompoundRisk(input)
	if rec == nil {
		t.Fatal("expected recommendation, got nil")
	}
	if rec.Priority != "high" {
		t.Errorf("expected priority high, got %s", rec.Priority)
	}
}

func TestEvalCompoundRisk_NoBlockers(t *testing.T) {
	input := CompoundRiskInput{
		EmployeeID:     "emp-id",
		EmployeeName:   "Dave",
		SentimentTrend: []string{"negative", "negative", "neutral"},
		ActiveBlockers: []string{},
	}
	rec := EvalCompoundRisk(input)
	if rec != nil {
		t.Error("expected nil when no active blockers")
	}
}

func TestEvalCompoundRisk_NoDecline(t *testing.T) {
	input := CompoundRiskInput{
		EmployeeID:     "emp-id",
		EmployeeName:   "Dave",
		SentimentTrend: []string{"positive", "positive", "neutral"},
		ActiveBlockers: []string{"tooling"},
	}
	rec := EvalCompoundRisk(input)
	if rec != nil {
		t.Error("expected nil when sentiment not declining")
	}
}

func TestEvalCompoundRisk_TooFewDataPoints(t *testing.T) {
	input := CompoundRiskInput{
		EmployeeID:     "emp-id",
		EmployeeName:   "Dave",
		SentimentTrend: []string{"negative", "neutral"},
		ActiveBlockers: []string{"tooling"},
	}
	rec := EvalCompoundRisk(input)
	if rec != nil {
		t.Error("expected nil when less than 3 sentiment data points")
	}
}

func TestIsDecliningSentiment(t *testing.T) {
	tests := []struct {
		name  string
		trend []string
		want  bool
	}{
		{"declining", []string{"negative", "neutral", "positive"}, true},
		{"flat negative", []string{"negative", "negative", "negative"}, false},
		{"improving", []string{"positive", "neutral", "negative"}, false},
		{"too few points", []string{"negative", "positive"}, false},
		{"mixed", []string{"negative", "positive", "neutral"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDecliningSentiment(tt.trend)
			if got != tt.want {
				t.Errorf("isDecliningSentiment(%v) = %v, want %v", tt.trend, got, tt.want)
			}
		})
	}
}

// Ensure RecommendationInput is compatible with brain package
func TestRecommendationInputType(t *testing.T) {
	rec := EvalBlockerEscalation(BlockerEscalationInput{
		EmployeeID: "id", EmployeeName: "X", Category: "y", Description: "z",
		RecurrenceCount: 5, FirstSeenAt: "2026-01-01",
	})
	var _ *brain.RecommendationInput = rec
}
