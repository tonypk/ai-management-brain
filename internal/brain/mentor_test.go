package brain_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestLoadMentor_Inamori(t *testing.T) {
	m, err := brain.LoadMentor("inamori")
	if err != nil {
		t.Fatalf("load inamori: %v", err)
	}
	if m.ID != "inamori" {
		t.Errorf("id = %q", m.ID)
	}
	if len(m.Strategy.CheckinQuestions) == 0 {
		t.Error("no checkin questions")
	}
	if m.Strategy.Chase.Method != "private_first" {
		t.Errorf("chase method = %q, want private_first", m.Strategy.Chase.Method)
	}
	if len(m.Strategy.Chase.Escalation) == 0 {
		t.Error("no escalation steps")
	}
	if m.Strategy.Summary.Highlight == "" {
		t.Error("no summary highlight")
	}
	if m.Strategy.SystemPrompt == "" {
		t.Error("no system prompt")
	}
}

func TestLoadMentor_Dalio(t *testing.T) {
	m, err := brain.LoadMentor("dalio")
	if err != nil {
		t.Fatalf("load dalio: %v", err)
	}
	if m.Strategy.Chase.Method != "public_direct" {
		t.Errorf("chase method = %q, want public_direct", m.Strategy.Chase.Method)
	}
}

func TestLoadMentor_NotFound(t *testing.T) {
	_, err := brain.LoadMentor("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent mentor")
	}
}

func TestGetCheckinQuestions(t *testing.T) {
	m, _ := brain.LoadMentor("inamori")
	qs := m.GetCheckinQuestions()
	if len(qs) < 2 {
		t.Errorf("expected at least 2 questions, got %d", len(qs))
	}
}

func TestGetChaseStep(t *testing.T) {
	m, _ := brain.LoadMentor("inamori")
	step1 := m.GetChaseStep(1)
	if step1.Action != "private_message" {
		t.Errorf("step 1 action = %q", step1.Action)
	}
	step2 := m.GetChaseStep(2)
	if step2.Action != "manager_notify" {
		t.Errorf("step 2 action = %q", step2.Action)
	}
	// Out of range returns skip
	stepN := m.GetChaseStep(99)
	if stepN.Action != "skip_today" {
		t.Errorf("out of range should return skip_today, got %q", stepN.Action)
	}
}
