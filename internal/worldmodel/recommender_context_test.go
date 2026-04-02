package worldmodel

import (
	"strings"
	"testing"
)

func TestSiloEntry_String(t *testing.T) {
	entry := SiloEntry{
		SkillName:    "payment_module",
		EmployeeName: "Alice",
		Confidence:   0.85,
	}
	text := entry.String()
	if text == "" {
		t.Error("expected non-empty string")
	}
	for _, want := range []string{"payment_module", "Alice", "85%"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestEscalatingBlockerInfo_String(t *testing.T) {
	info := EscalatingBlockerInfo{
		EmployeeName:    "Bob",
		Category:        "tooling",
		Description:     "CI keeps breaking",
		RecurrenceCount: 4,
		FirstSeenAt:     "2026-03-15",
	}
	text := info.String()
	for _, want := range []string{"Bob", "tooling", "CI keeps breaking", "x4", "2026-03-15"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestGrowthSignalInfo_String(t *testing.T) {
	info := GrowthSignalInfo{
		EmployeeName: "Carol",
		EventType:    "skill_upgrade",
		Description:  "Upgraded Go proficiency",
	}
	text := info.String()
	for _, want := range []string{"Carol", "skill_upgrade", "Upgraded Go proficiency"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestRiskInsightInfo_String(t *testing.T) {
	info := RiskInsightInfo{
		Dimension:   "risk",
		InsightText: "Team velocity declining",
		Confidence:  0.75,
	}
	text := info.String()
	for _, want := range []string{"risk", "Team velocity declining", "75%"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestFormatForPrompt_WithData(t *testing.T) {
	rc := &RecommenderContext{
		KnowledgeSilos: []SiloEntry{
			{SkillName: "payment", EmployeeName: "Alice", Confidence: 0.9},
		},
		EscalatingBlockers: []EscalatingBlockerInfo{
			{EmployeeName: "Bob", Category: "tooling", Description: "CI keeps breaking", RecurrenceCount: 4, FirstSeenAt: "2026-03-15"},
		},
		GrowthSignals: []GrowthSignalInfo{
			{EmployeeName: "Carol", EventType: "new_skill", Description: "Learned testing"},
		},
		RiskInsights: []RiskInsightInfo{
			{Dimension: "risk", InsightText: "Velocity drop", Confidence: 0.8},
		},
	}
	text := rc.FormatForPrompt()
	if text == "" {
		t.Error("expected non-empty text")
	}
	for _, want := range []string{
		"Knowledge Silo", "payment", "Alice",
		"Escalating Blocker", "Bob", "tooling",
		"Growth Events", "Carol", "new_skill",
		"Risk & Opportunity", "Velocity drop",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in output, got: %s", want, text)
		}
	}
}

func TestFormatForPrompt_Empty(t *testing.T) {
	rc := &RecommenderContext{}
	text := rc.FormatForPrompt()
	if text != "" {
		t.Errorf("expected empty string for empty context, got: %s", text)
	}
}

func TestFormatForPrompt_PartialData(t *testing.T) {
	rc := &RecommenderContext{
		KnowledgeSilos: []SiloEntry{
			{SkillName: "go_api", EmployeeName: "Dave", Confidence: 0.85},
		},
	}
	text := rc.FormatForPrompt()
	if !strings.Contains(text, "Knowledge Silo") {
		t.Error("expected Knowledge Silo section")
	}
	if strings.Contains(text, "Escalating Blocker") {
		t.Error("should not contain Escalating Blockers when empty")
	}
}
