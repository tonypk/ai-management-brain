package brain_test

import (
	"context"
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
	// Mock LLM that returns a valid JSON config
	mockResp := `{
		"id": "jobs",
		"name": "Steve Jobs",
		"name_en": "Steve Jobs",
		"company": "Apple",
		"philosophy": "Stay hungry, stay foolish",
		"checkin_questions": ["What did you ship today?", "What's blocking you?", "What would you simplify?"],
		"chase_method": "private_first",
		"chase_escalation": [
			{"action": "private_message", "delay": "0", "tone": "direct_challenge"},
			{"action": "manager_notify", "delay": "1h", "tone": "urgent"},
			{"action": "skip_today", "delay": "2h"}
		],
		"chase_forbidden": ["long_delay"],
		"chase_encouraged": ["direct_feedback"],
		"summary_focus": ["innovation", "execution", "quality"],
		"summary_highlight": "breakthroughs",
		"summary_flag": "mediocrity",
		"summary_metrics": [{"name": "Ship Rate", "source": "task_completion"}],
		"weekly_actions": [{"type": "review", "desc": "Product review"}],
		"monthly_actions": [{"type": "vision", "desc": "Vision alignment"}],
		"triggers": [
			{"event": "consecutive_miss_3days", "action": "direct_call", "message": "{name} needs accountability"},
			{"event": "sentiment_drop", "action": "one_on_one", "message": "Check on {name}"}
		],
		"system_prompt": "You embody Steve Jobs management style. Focus on simplicity and excellence."
	}`
	llm := &mockLLM{response: mockResp}
	gen := brain.NewMentorGenerator(llm)

	result, err := gen.Generate(context.Background(), brain.CustomMentorRequest{Name: "Steve Jobs"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Config == nil {
		t.Fatal("expected non-nil config")
	}
	if result.Config.ID != "jobs" {
		t.Errorf("id = %q, want %q", result.Config.ID, "jobs")
	}
	if result.Config.NameEn != "Steve Jobs" {
		t.Errorf("name_en = %q, want %q", result.Config.NameEn, "Steve Jobs")
	}
	if len(result.Config.Strategy.CheckinQuestions) != 3 {
		t.Errorf("expected 3 checkin questions, got %d", len(result.Config.Strategy.CheckinQuestions))
	}
	if result.FilePath == "" {
		t.Error("expected non-empty file path")
	}

	// Verify the mentor was registered as valid
	if !brain.ValidMentors["jobs"] {
		t.Error("expected 'jobs' to be registered as valid mentor")
	}

	// Clean up: remove the generated file and unregister
	delete(brain.ValidMentors, "jobs")
}
