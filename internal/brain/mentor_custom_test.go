package brain_test

import (
	"context"
	"os"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestMentorGenerator_RequiresName(t *testing.T) {
	gen := brain.NewMentorGenerator(nil)
	_, err := gen.Generate(context.Background(), brain.CustomMentorRequest{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestMentorGenerator_RequiresLLM(t *testing.T) {
	gen := brain.NewMentorGenerator(nil)
	_, err := gen.Generate(context.Background(), brain.CustomMentorRequest{Name: "Steve Jobs"})
	if err == nil {
		t.Error("expected error when LLM is nil")
	}
}

func TestMentorGenerator_GeneratesConfig(t *testing.T) {
	// Mock LLM that returns a valid JSON config — use a unique test ID to avoid overwriting real configs
	mockResp := `{
		"id": "test_mentor_xyz",
		"name": "Test Mentor",
		"name_en": "Test Mentor",
		"company": "TestCo",
		"philosophy": "Test philosophy",
		"checkin_questions": ["Q1?", "Q2?", "Q3?"],
		"chase_method": "private_first",
		"chase_escalation": [
			{"action": "private_message", "delay": "0", "tone": "warm"},
			{"action": "skip_today", "delay": "2h"}
		],
		"chase_forbidden": ["harsh"],
		"chase_encouraged": ["kind"],
		"summary_focus": ["quality", "speed"],
		"summary_highlight": "wins",
		"summary_flag": "blockers",
		"summary_metrics": [{"name": "Test Metric", "source": "test_source"}],
		"weekly_actions": [{"type": "review", "desc": "Weekly review"}],
		"monthly_actions": [{"type": "report", "desc": "Monthly report"}],
		"triggers": [
			{"event": "consecutive_miss_3days", "action": "notify", "message": "{name} missed"}
		],
		"system_prompt": "Test system prompt."
	}`
	llm := &mockLLM{response: mockResp}
	gen := brain.NewMentorGenerator(llm)

	result, err := gen.Generate(context.Background(), brain.CustomMentorRequest{Name: "Test Mentor"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Config == nil {
		t.Fatal("expected non-nil config")
	}
	if result.Config.ID != "test_mentor_xyz" {
		t.Errorf("id = %q, want %q", result.Config.ID, "test_mentor_xyz")
	}
	if len(result.Config.Strategy.CheckinQuestions) != 3 {
		t.Errorf("expected 3 checkin questions, got %d", len(result.Config.Strategy.CheckinQuestions))
	}
	if result.FilePath == "" {
		t.Error("expected non-empty file path")
	}

	// Verify the mentor was registered as valid
	if !brain.ValidMentors["test_mentor_xyz"] {
		t.Error("expected 'test_mentor_xyz' to be registered as valid mentor")
	}

	// Clean up: remove the generated file and unregister
	os.Remove(result.FilePath)
	delete(brain.ValidMentors, "test_mentor_xyz")
}
