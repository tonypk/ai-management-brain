package brain_test

import (
	"strings"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// ---------------------------------------------------------------------------
// ForBlend (EngineFactory)
// ---------------------------------------------------------------------------

func TestEngineFactory_ForBlend_Basic(t *testing.T) {
	f := brain.NewEngineFactory()
	e, err := f.ForBlend("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("ForBlend: %v", err)
	}
	if !strings.Contains(e.MentorID(), "inamori") {
		t.Errorf("expected blended ID to contain 'inamori', got %q", e.MentorID())
	}
	if !strings.Contains(e.MentorID(), "dalio") {
		t.Errorf("expected blended ID to contain 'dalio', got %q", e.MentorID())
	}
}

func TestEngineFactory_ForBlend_Caching(t *testing.T) {
	f := brain.NewEngineFactory()
	e1, err := f.ForBlend("inamori", "grove", 0.7, "default")
	if err != nil {
		t.Fatalf("first ForBlend: %v", err)
	}
	e2, err := f.ForBlend("inamori", "grove", 0.7, "default")
	if err != nil {
		t.Fatalf("second ForBlend: %v", err)
	}
	if e1 != e2 {
		t.Error("ForBlend should return cached engine for same parameters")
	}
}

func TestEngineFactory_ForBlend_DifferentWeights_NotCached(t *testing.T) {
	f := brain.NewEngineFactory()
	e1, _ := f.ForBlend("inamori", "dalio", 0.7, "default")
	e2, _ := f.ForBlend("inamori", "dalio", 0.5, "default")
	if e1 == e2 {
		t.Error("ForBlend with different weights should return different engines")
	}
}

func TestEngineFactory_ForBlend_DifferentCultures_NotCached(t *testing.T) {
	f := brain.NewEngineFactory()
	e1, _ := f.ForBlend("inamori", "dalio", 0.7, "default")
	e2, _ := f.ForBlend("inamori", "dalio", 0.7, "philippines")
	if e1 == e2 {
		t.Error("ForBlend with different cultures should return different engines")
	}
}

func TestEngineFactory_ForBlend_InvalidPrimary(t *testing.T) {
	f := brain.NewEngineFactory()
	_, err := f.ForBlend("nonexistent", "dalio", 0.7, "default")
	if err == nil {
		t.Error("ForBlend with invalid primary mentor should return error")
	}
}

func TestEngineFactory_ForBlend_InvalidSecondary(t *testing.T) {
	f := brain.NewEngineFactory()
	_, err := f.ForBlend("inamori", "nonexistent", 0.7, "default")
	if err == nil {
		t.Error("ForBlend with invalid secondary mentor should return error")
	}
}

// ---------------------------------------------------------------------------
// NewBlendedEngine
// ---------------------------------------------------------------------------

func TestNewBlendedEngine_MergedID(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	expected := "inamori+dalio"
	if e.MentorID() != expected {
		t.Errorf("expected blended ID %q, got %q", expected, e.MentorID())
	}
}

func TestNewBlendedEngine_QuestionsIncludeSecondary(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	qs := e.GetCheckinQuestions()
	// Primary questions + 1 from secondary
	if len(qs) < 3 {
		t.Errorf("expected at least 3 blended questions, got %d", len(qs))
	}
}

func TestNewBlendedEngine_SystemPromptContainsBoth(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	prompt := e.BuildSystemPrompt()
	if !strings.Contains(prompt, "Secondary Mentor Influence") {
		t.Error("blended prompt should contain 'Secondary Mentor Influence' marker")
	}
	if !strings.Contains(prompt, "30%") {
		t.Error("blended prompt should show secondary weight percentage (30%)")
	}
}

func TestNewBlendedEngine_SummaryFocusMerged(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	cfg := e.GetSummaryConfig()
	if len(cfg.Focus) < 3 {
		t.Errorf("expected at least 3 merged focus areas, got %d", len(cfg.Focus))
	}
}

func TestNewBlendedEngine_SummaryFocusDeduplicates(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "inamori", 0.5, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	cfg := e.GetSummaryConfig()
	seen := make(map[string]bool)
	for _, f := range cfg.Focus {
		if seen[f] {
			t.Errorf("duplicate focus area found: %q", f)
		}
		seen[f] = true
	}
}

func TestNewBlendedEngine_MetricsMerged(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	cfg := e.GetSummaryConfig()
	if len(cfg.Metrics) < 2 {
		t.Errorf("expected at least 2 merged metrics, got %d", len(cfg.Metrics))
	}
}

func TestNewBlendedEngine_MetricsDeduplicates(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "inamori", 0.5, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	cfg := e.GetSummaryConfig()
	seen := make(map[string]bool)
	for _, m := range cfg.Metrics {
		if seen[m.Name] {
			t.Errorf("duplicate metric found: %q", m.Name)
		}
		seen[m.Name] = true
	}
}

func TestNewBlendedEngine_TriggersMerged(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	triggers := e.GetTriggerRules()
	if len(triggers) < 2 {
		t.Errorf("expected at least 2 merged triggers, got %d", len(triggers))
	}
}

func TestNewBlendedEngine_TriggersDeduplicateByEvent(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "inamori", 0.5, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	triggers := e.GetTriggerRules()
	seen := make(map[string]bool)
	for _, tr := range triggers {
		if seen[tr.Event] {
			t.Errorf("duplicate trigger event found: %q", tr.Event)
		}
		seen[tr.Event] = true
	}
}

func TestNewBlendedEngine_CultureOverrideStillApplies(t *testing.T) {
	e, err := brain.NewBlendedEngine("dalio", "grove", 0.7, "philippines")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	step := e.GetEffectiveChaseStep(1)
	if step.Action != "private_message" {
		t.Errorf("PH culture should override blended chase to private_message, got %q", step.Action)
	}
}

func TestNewBlendedEngine_InvalidPrimary(t *testing.T) {
	_, err := brain.NewBlendedEngine("nonexistent", "dalio", 0.7, "default")
	if err == nil {
		t.Error("NewBlendedEngine with invalid primary should return error")
	}
	if !strings.Contains(err.Error(), "primary") {
		t.Errorf("error should mention 'primary', got: %v", err)
	}
}

func TestNewBlendedEngine_InvalidSecondary(t *testing.T) {
	_, err := brain.NewBlendedEngine("inamori", "nonexistent", 0.7, "default")
	if err == nil {
		t.Error("NewBlendedEngine with invalid secondary should return error")
	}
	if !strings.Contains(err.Error(), "secondary") {
		t.Errorf("error should mention 'secondary', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetSummaryConfig
// ---------------------------------------------------------------------------

func TestEngine_GetSummaryConfig_Inamori(t *testing.T) {
	e, err := brain.NewEngine("inamori", "default")
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	cfg := e.GetSummaryConfig()
	if len(cfg.Focus) == 0 {
		t.Error("expected at least one focus area")
	}
	if cfg.Highlight == "" {
		t.Error("expected non-empty highlight")
	}
	if cfg.Flag == "" {
		t.Error("expected non-empty flag")
	}
}

func TestEngine_GetSummaryConfig_AllMentors(t *testing.T) {
	mentors := []string{"inamori", "dalio", "grove", "ren", "son", "jobs", "bezos", "ma"}
	for _, m := range mentors {
		t.Run(m, func(t *testing.T) {
			e, err := brain.NewEngine(m, "default")
			if err != nil {
				t.Fatalf("NewEngine(%q): %v", m, err)
			}
			cfg := e.GetSummaryConfig()
			if len(cfg.Focus) == 0 {
				t.Error("summary should have at least one focus area")
			}
			if cfg.Highlight == "" {
				t.Error("summary should have a highlight")
			}
			if cfg.Flag == "" {
				t.Error("summary should have a flag")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetTriggerRules
// ---------------------------------------------------------------------------

func TestEngine_GetTriggerRules_AllMentors(t *testing.T) {
	mentors := []string{"inamori", "dalio", "grove", "ren", "son", "jobs", "bezos", "ma"}
	for _, m := range mentors {
		t.Run(m, func(t *testing.T) {
			e, err := brain.NewEngine(m, "default")
			if err != nil {
				t.Fatalf("NewEngine(%q): %v", m, err)
			}
			triggers := e.GetTriggerRules()
			if len(triggers) == 0 {
				t.Error("every mentor should have at least one trigger rule")
			}
			for i, tr := range triggers {
				if tr.Event == "" {
					t.Errorf("trigger[%d] has empty event", i)
				}
				if tr.Action == "" {
					t.Errorf("trigger[%d] has empty action", i)
				}
				if tr.Message == "" {
					t.Errorf("trigger[%d] has empty message", i)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetWeeklyActions / GetMonthlyActions
// ---------------------------------------------------------------------------

func TestEngine_GetWeeklyActions_AllMentors(t *testing.T) {
	mentors := []string{"inamori", "dalio", "grove", "ren", "son", "jobs", "bezos", "ma"}
	for _, m := range mentors {
		t.Run(m, func(t *testing.T) {
			e, err := brain.NewEngine(m, "default")
			if err != nil {
				t.Fatalf("NewEngine(%q): %v", m, err)
			}
			actions := e.GetWeeklyActions()
			if len(actions) == 0 {
				t.Error("every mentor should have at least one weekly action")
			}
			for i, a := range actions {
				if a.Type == "" {
					t.Errorf("weekly[%d] has empty type", i)
				}
				if a.Desc == "" {
					t.Errorf("weekly[%d] has empty desc", i)
				}
			}
		})
	}
}

func TestEngine_GetMonthlyActions_AllMentors(t *testing.T) {
	mentors := []string{"inamori", "dalio", "grove", "ren", "son", "jobs", "bezos", "ma"}
	for _, m := range mentors {
		t.Run(m, func(t *testing.T) {
			e, err := brain.NewEngine(m, "default")
			if err != nil {
				t.Fatalf("NewEngine(%q): %v", m, err)
			}
			actions := e.GetMonthlyActions()
			if len(actions) == 0 {
				t.Error("every mentor should have at least one monthly action")
			}
			for i, a := range actions {
				if a.Type == "" {
					t.Errorf("monthly[%d] has empty type", i)
				}
				if a.Desc == "" {
					t.Errorf("monthly[%d] has empty desc", i)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Blend + weekly/monthly
// ---------------------------------------------------------------------------

func TestNewBlendedEngine_WeeklyActionsFromPrimary(t *testing.T) {
	e, err := brain.NewBlendedEngine("inamori", "dalio", 0.7, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	actions := e.GetWeeklyActions()
	if len(actions) == 0 {
		t.Fatal("blended engine should have weekly actions")
	}
}

func TestNewBlendedEngine_MonthlyActionsFromPrimary(t *testing.T) {
	e, err := brain.NewBlendedEngine("grove", "ren", 0.6, "default")
	if err != nil {
		t.Fatalf("NewBlendedEngine: %v", err)
	}
	actions := e.GetMonthlyActions()
	if len(actions) == 0 {
		t.Fatal("blended engine should have monthly actions")
	}
}

func TestNewBlendedEngine_AllMentorPairs(t *testing.T) {
	pairs := []struct {
		primary   string
		secondary string
	}{
		{"inamori", "dalio"},
		{"grove", "ren"},
		{"dalio", "grove"},
		{"ren", "inamori"},
		{"son", "jobs"},
		{"bezos", "ma"},
	}
	for _, p := range pairs {
		name := p.primary + "+" + p.secondary
		t.Run(name, func(t *testing.T) {
			e, err := brain.NewBlendedEngine(p.primary, p.secondary, 0.7, "default")
			if err != nil {
				t.Fatalf("NewBlendedEngine(%s, %s): %v", p.primary, p.secondary, err)
			}
			if e.MentorID() != p.primary+"+"+p.secondary {
				t.Errorf("expected mentor ID %q, got %q", p.primary+"+"+p.secondary, e.MentorID())
			}
			if len(e.GetCheckinQuestions()) == 0 {
				t.Error("blended engine should have checkin questions")
			}
			if len(e.GetSummaryConfig().Focus) == 0 {
				t.Error("blended engine should have summary focus areas")
			}
			if len(e.GetTriggerRules()) == 0 {
				t.Error("blended engine should have trigger rules")
			}
			if len(e.GetWeeklyActions()) == 0 {
				t.Error("blended engine should have weekly actions")
			}
			if len(e.GetMonthlyActions()) == 0 {
				t.Error("blended engine should have monthly actions")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetChaseStep edge cases
// ---------------------------------------------------------------------------

func TestEngine_GetChaseStep_OutOfBounds(t *testing.T) {
	e, _ := brain.NewEngine("inamori", "default")
	step := e.GetEffectiveChaseStep(0)
	if step.Action != "skip_today" {
		t.Errorf("step 0 should return skip_today, got %q", step.Action)
	}
	step = e.GetEffectiveChaseStep(999)
	if step.Action != "skip_today" {
		t.Errorf("step 999 should return skip_today, got %q", step.Action)
	}
}

func TestEngine_GetChaseStep_NegativeIndex(t *testing.T) {
	e, _ := brain.NewEngine("dalio", "default")
	step := e.GetEffectiveChaseStep(-1)
	if step.Action != "skip_today" {
		t.Errorf("step -1 should return skip_today, got %q", step.Action)
	}
}
