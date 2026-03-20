package brain_test

import (
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestEngine_BuildSystemPrompt(t *testing.T) {
	e, err := brain.NewEngine("inamori", "philippines")
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	prompt := e.BuildSystemPrompt()
	if prompt == "" {
		t.Fatal("empty prompt")
	}
	// Should contain mentor philosophy
	if !strings.Contains(prompt, "利他") {
		t.Error("prompt should contain Inamori's philosophy")
	}
	// Should contain culture rules
	if !strings.Contains(prompt, "Philippines") {
		t.Error("prompt should include cultural context")
	}
	// Should contain forbidden patterns
	if !strings.Contains(prompt, "FORBIDDEN") {
		t.Error("prompt should include forbidden section")
	}
}

func TestEngine_GetChaseMessage_CultureOverride(t *testing.T) {
	// Dalio wants public, but PH culture should force private
	e, _ := brain.NewEngine("dalio", "philippines")
	step := e.GetEffectiveChaseStep(1)
	// PH culture should override Dalio's public_reminder to private_message
	if step.Action == "public_reminder" {
		t.Error("PH culture should override Dalio's public chase to private")
	}
	if step.Action != "private_message" {
		t.Errorf("expected private_message, got %q", step.Action)
	}
}

func TestEngine_GetEffectiveChaseStep_NoOverride(t *testing.T) {
	// Dalio + SG = no override needed (both direct)
	e, _ := brain.NewEngine("dalio", "singapore")
	step := e.GetEffectiveChaseStep(1)
	if step.Action != "public_reminder" {
		t.Errorf("SG should not override Dalio's public chase, got %q", step.Action)
	}
}

func TestEngine_GetCheckinQuestions(t *testing.T) {
	e, _ := brain.NewEngine("inamori", "philippines")
	qs := e.GetCheckinQuestions()
	if len(qs) < 2 {
		t.Errorf("expected at least 2 questions, got %d", len(qs))
	}
}
