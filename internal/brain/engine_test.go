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

func TestEngineFactory_Caching(t *testing.T) {
	f := brain.NewEngineFactory()
	e1, err := f.ForTenant("inamori", "default")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	e2, err := f.ForTenant("inamori", "default")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if e1 != e2 {
		t.Error("factory should return cached engine")
	}
}

func TestEngineFactory_DifferentMentors(t *testing.T) {
	f := brain.NewEngineFactory()
	mentors := []string{"inamori", "dalio", "grove", "ren"}
	for _, m := range mentors {
		e, err := f.ForTenant(m, "default")
		if err != nil {
			t.Fatalf("failed for mentor %s: %v", m, err)
		}
		if e.MentorID() != m {
			t.Errorf("expected mentor %s, got %s", m, e.MentorID())
		}
	}
}

func TestEngineFactory_DifferentCultures(t *testing.T) {
	f := brain.NewEngineFactory()
	cultures := []string{"default", "philippines", "singapore", "indonesia", "srilanka"}
	for _, c := range cultures {
		_, err := f.ForTenant("inamori", c)
		if err != nil {
			t.Fatalf("failed for culture %s: %v", c, err)
		}
	}
}

func TestEngineFactory_Invalidate(t *testing.T) {
	f := brain.NewEngineFactory()
	e1, _ := f.ForTenant("inamori", "default")
	f.Invalidate("inamori", "default")
	e2, _ := f.ForTenant("inamori", "default")
	if e1 == e2 {
		t.Error("after invalidate, should return new engine")
	}
}

func TestValidMentors(t *testing.T) {
	expected := []string{"inamori", "dalio", "grove", "ren"}
	for _, m := range expected {
		if !brain.ValidMentors[m] {
			t.Errorf("expected %s in ValidMentors", m)
		}
	}
}

func TestEngine_MentorID(t *testing.T) {
	e, _ := brain.NewEngine("grove", "default")
	if e.MentorID() != "grove" {
		t.Errorf("expected grove, got %s", e.MentorID())
	}
}
