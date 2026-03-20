package brain_test

import (
	"context"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestOrchestrator_AskEmployee(t *testing.T) {
	orch := brain.NewOrchestrator(nil)
	intent, err := orch.DetectIntent(context.Background(), "Ask John about project progress")
	if err != nil {
		t.Fatalf("DetectIntent: %v", err)
	}
	if intent.Type != brain.IntentAskEmployee {
		t.Errorf("type = %q, want %q", intent.Type, brain.IntentAskEmployee)
	}
	if intent.Target != "john" {
		t.Errorf("target = %q, want %q", intent.Target, "john")
	}
	if intent.Content != "project progress" {
		t.Errorf("content = %q, want %q", intent.Content, "project progress")
	}
}

func TestOrchestrator_Announce(t *testing.T) {
	orch := brain.NewOrchestrator(nil)

	tests := []struct {
		input   string
		content string
	}{
		{"Tell the team meeting at 3pm", "meeting at 3pm"},
		{"Announce holiday tomorrow", "holiday tomorrow"},
	}

	for _, tt := range tests {
		intent, err := orch.DetectIntent(context.Background(), tt.input)
		if err != nil {
			t.Fatalf("DetectIntent(%q): %v", tt.input, err)
		}
		if intent.Type != brain.IntentAnnounce {
			t.Errorf("type for %q = %q, want %q", tt.input, intent.Type, brain.IntentAnnounce)
		}
	}
}

func TestOrchestrator_CheckStatus(t *testing.T) {
	orch := brain.NewOrchestrator(nil)

	tests := []struct {
		input  string
		target string
	}{
		{"How is Alice doing?", "alice"},
		{"Status of Bob", "bob"},
	}

	for _, tt := range tests {
		intent, err := orch.DetectIntent(context.Background(), tt.input)
		if err != nil {
			t.Fatalf("DetectIntent(%q): %v", tt.input, err)
		}
		if intent.Type != brain.IntentCheckStatus {
			t.Errorf("type for %q = %q, want %q", tt.input, intent.Type, brain.IntentCheckStatus)
		}
		if intent.Target != tt.target {
			t.Errorf("target for %q = %q, want %q", tt.input, intent.Target, tt.target)
		}
	}
}

func TestOrchestrator_SwitchMentor(t *testing.T) {
	orch := brain.NewOrchestrator(nil)

	intent, err := orch.DetectIntent(context.Background(), "Switch to dalio")
	if err != nil {
		t.Fatalf("DetectIntent: %v", err)
	}
	if intent.Type != brain.IntentSwitchMentor {
		t.Errorf("type = %q, want %q", intent.Type, brain.IntentSwitchMentor)
	}
	if intent.Target != "dalio" {
		t.Errorf("target = %q, want %q", intent.Target, "dalio")
	}
}

func TestOrchestrator_GetSummary(t *testing.T) {
	orch := brain.NewOrchestrator(nil)

	tests := []string{"summary", "Give me today's summary", "daily summary please"}
	for _, input := range tests {
		intent, err := orch.DetectIntent(context.Background(), input)
		if err != nil {
			t.Fatalf("DetectIntent(%q): %v", input, err)
		}
		if intent.Type != brain.IntentGetSummary {
			t.Errorf("type for %q = %q, want %q", input, intent.Type, brain.IntentGetSummary)
		}
	}
}

func TestOrchestrator_Unknown(t *testing.T) {
	orch := brain.NewOrchestrator(nil) // no LLM = falls through to unknown
	intent, err := orch.DetectIntent(context.Background(), "xyzzy foo bar")
	if err != nil {
		t.Fatalf("DetectIntent: %v", err)
	}
	if intent.Type != brain.IntentUnknown {
		t.Errorf("type = %q, want %q", intent.Type, brain.IntentUnknown)
	}
}

func TestOrchestrator_Dispatch_AskEmployee(t *testing.T) {
	orch := brain.NewOrchestrator(nil)
	intent := brain.Intent{
		Type:    brain.IntentAskEmployee,
		Target:  "john",
		Content: "how is the project?",
	}
	task := orch.Dispatch(intent)
	if task.Action != "send_question" {
		t.Errorf("action = %q, want %q", task.Action, "send_question")
	}
	if task.Params["target"] != "john" {
		t.Errorf("params[target] = %q, want %q", task.Params["target"], "john")
	}
}

func TestOrchestrator_Dispatch_Unknown(t *testing.T) {
	orch := brain.NewOrchestrator(nil)
	intent := brain.Intent{Type: brain.IntentUnknown}
	task := orch.Dispatch(intent)
	if task.Status != "failed" {
		t.Errorf("status = %q, want %q", task.Status, "failed")
	}
}

func TestOrchestrator_SetReminder(t *testing.T) {
	orch := brain.NewOrchestrator(nil)
	intent, err := orch.DetectIntent(context.Background(), "Remind me to review reports")
	if err != nil {
		t.Fatalf("DetectIntent: %v", err)
	}
	if intent.Type != brain.IntentSetReminder {
		t.Errorf("type = %q, want %q", intent.Type, brain.IntentSetReminder)
	}
	if intent.Content != "review reports" {
		t.Errorf("content = %q, want %q", intent.Content, "review reports")
	}
}
