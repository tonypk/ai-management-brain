package onboarding

import (
	"strings"
	"testing"
)

func TestBuildConsultantPrompt_EmptyData(t *testing.T) {
	collected := &CollectedData{}
	result := BuildConsultantPrompt(collected, 0)

	if !strings.Contains(result, "Nothing collected yet") {
		t.Error("expected 'Nothing collected yet' for empty data")
	}

	// Should list all 8 required missing fields
	requiredFields := []string{
		"Industry",
		"Company stage",
		"Business model",
		"Team size",
		"Organizational structure",
		"Current projects",
		"Management pain points",
		"Communication tools",
	}
	for _, field := range requiredFields {
		if !strings.Contains(result, field) {
			t.Errorf("expected missing field %q in prompt", field)
		}
	}

	// Should contain the system instruction preamble
	if !strings.Contains(result, "management consultant") {
		t.Error("expected system instruction preamble")
	}

	// Should NOT contain wrapping up text at turn 0
	if strings.Contains(result, "wrapping up") {
		t.Error("should not contain wrapping up text at turn 0")
	}
}

func TestBuildConsultantPrompt_PartialData(t *testing.T) {
	collected := &CollectedData{
		Industry:     "SaaS",
		TeamSize:     25,
		CompanyStage: "Series A",
		PainPoints:   []string{"hiring", "retention"},
	}
	result := BuildConsultantPrompt(collected, 5)

	// Collected fields should appear
	if !strings.Contains(result, "Industry: SaaS") {
		t.Error("expected collected Industry in prompt")
	}
	if !strings.Contains(result, "Team size: 25") {
		t.Error("expected collected TeamSize in prompt")
	}
	if !strings.Contains(result, "Company stage: Series A") {
		t.Error("expected collected CompanyStage in prompt")
	}
	if !strings.Contains(result, "hiring, retention") {
		t.Error("expected collected PainPoints in prompt")
	}

	// Should NOT contain "Nothing collected yet"
	if strings.Contains(result, "Nothing collected yet") {
		t.Error("should not contain 'Nothing collected yet' when data exists")
	}

	// Missing fields should still appear
	missingFields := []string{
		"Business model",
		"Organizational structure",
		"Current projects",
		"Communication tools",
	}
	for _, field := range missingFields {
		if !strings.Contains(result, field) {
			t.Errorf("expected missing field %q in prompt", field)
		}
	}

	// Fields that ARE collected should NOT appear in "Still Need" section
	stillNeed := result[strings.Index(result, "## Still Need"):]
	if strings.Contains(stillNeed, "Industry") {
		t.Error("Industry should not appear in Still Need section")
	}
}

func TestBuildConsultantPrompt_AllDataCollected(t *testing.T) {
	collected := &CollectedData{
		Industry:        "Fintech",
		CompanyStage:    "Growth",
		BusinessModel:   "B2B SaaS",
		TeamSize:        50,
		OrgStructure:    "Matrix",
		CurrentProjects: "Platform migration",
		PainPoints:      []string{"scaling"},
		CommTools:       []string{"Slack", "Zoom"},
		CulturePrefs:    "Collaborative",
		GoalFramework:   "OKR",
	}
	result := BuildConsultantPrompt(collected, 10)

	if !strings.Contains(result, "ALL REQUIRED INFO COLLECTED") {
		t.Error("expected 'ALL REQUIRED INFO COLLECTED' when all fields present")
	}

	// Optional fields should also be listed
	if !strings.Contains(result, "Culture prefs: Collaborative") {
		t.Error("expected CulturePrefs in collected section")
	}
	if !strings.Contains(result, "Goal framework: OKR") {
		t.Error("expected GoalFramework in collected section")
	}
}

func TestBuildConsultantPrompt_WrappingUpAt15(t *testing.T) {
	collected := &CollectedData{}
	result := BuildConsultantPrompt(collected, 15)

	if !strings.Contains(result, "wrapping up") {
		t.Error("expected 'wrapping up' text at messageCount 15")
	}
	if !strings.Contains(result, "15/20") {
		t.Error("expected turn count '15/20' in prompt")
	}

	// Should NOT contain TURN LIMIT at 15
	if strings.Contains(result, "TURN LIMIT REACHED") {
		t.Error("should not contain 'TURN LIMIT REACHED' at turn 15")
	}
}

func TestBuildConsultantPrompt_WrappingUpAt18(t *testing.T) {
	collected := &CollectedData{}
	result := BuildConsultantPrompt(collected, 18)

	if !strings.Contains(result, "wrapping up") {
		t.Error("expected 'wrapping up' text at messageCount 18")
	}
	if !strings.Contains(result, "18/20") {
		t.Error("expected turn count '18/20' in prompt")
	}
}

func TestBuildConsultantPrompt_TurnLimitAt20(t *testing.T) {
	collected := &CollectedData{}
	result := BuildConsultantPrompt(collected, 20)

	if !strings.Contains(result, "TURN LIMIT REACHED") {
		t.Error("expected 'TURN LIMIT REACHED' at messageCount 20")
	}
	// At 20, both >= 15 and >= 20 conditions trigger
	if !strings.Contains(result, "wrapping up") {
		t.Error("expected 'wrapping up' text at messageCount 20 (>= 15)")
	}
}

func TestBuildConsultantPrompt_TurnLimitAbove20(t *testing.T) {
	collected := &CollectedData{}
	result := BuildConsultantPrompt(collected, 25)

	if !strings.Contains(result, "TURN LIMIT REACHED") {
		t.Error("expected 'TURN LIMIT REACHED' at messageCount 25")
	}
	if !strings.Contains(result, "25/20") {
		t.Error("expected turn count '25/20' in prompt")
	}
}

func TestBuildExtractionPrompt(t *testing.T) {
	currentData := &CollectedData{
		Industry: "Healthcare",
		TeamSize: 10,
	}
	userMessage := "We use Slack and have about 30 people now"

	result := BuildExtractionPrompt(currentData, userMessage)

	// Should contain the current data as JSON
	if !strings.Contains(result, `"industry":"Healthcare"`) {
		t.Error("expected current industry in JSON")
	}
	if !strings.Contains(result, `"team_size":10`) {
		t.Error("expected current team_size in JSON")
	}

	// Should contain the user message
	if !strings.Contains(result, userMessage) {
		t.Error("expected user message in extraction prompt")
	}

	// Should contain extraction instructions
	if !strings.Contains(result, "Extract structured information") {
		t.Error("expected extraction instructions in prompt")
	}
	if !strings.Contains(result, "CollectedData") {
		t.Error("expected CollectedData schema reference")
	}
}

func TestBuildExtractionPrompt_EmptyData(t *testing.T) {
	currentData := &CollectedData{}
	userMessage := "We are a fintech startup"

	result := BuildExtractionPrompt(currentData, userMessage)

	// Empty CollectedData should marshal to "{}"
	if !strings.Contains(result, "Current data: {}") {
		t.Error("expected empty JSON for empty CollectedData")
	}
	if !strings.Contains(result, userMessage) {
		t.Error("expected user message in extraction prompt")
	}
}

func TestMissingRequired_AllFields(t *testing.T) {
	c := &CollectedData{
		Industry:        "Tech",
		CompanyStage:    "Startup",
		BusinessModel:   "B2C",
		TeamSize:        15,
		OrgStructure:    "Flat",
		CurrentProjects: "MVP launch",
		PainPoints:      []string{"communication"},
		CommTools:       []string{"Slack"},
	}
	missing := missingRequired(c)
	if len(missing) != 0 {
		t.Errorf("expected 0 missing fields, got %d: %v", len(missing), missing)
	}
}

func TestMissingRequired_NoFields(t *testing.T) {
	c := &CollectedData{}
	missing := missingRequired(c)
	if len(missing) != 8 {
		t.Errorf("expected 8 missing fields, got %d: %v", len(missing), missing)
	}

	// Verify the exact fields returned
	expected := []string{
		"Industry",
		"Company stage",
		"Business model",
		"Team size",
		"Organizational structure",
		"Current projects",
		"Management pain points",
		"Communication tools",
	}
	for i, field := range expected {
		if i >= len(missing) {
			break
		}
		if missing[i] != field {
			t.Errorf("missing[%d] = %q, want %q", i, missing[i], field)
		}
	}
}

func TestMissingRequired_PartialFields(t *testing.T) {
	c := &CollectedData{
		Industry:   "Retail",
		TeamSize:   5,
		PainPoints: []string{"inventory"},
	}
	missing := missingRequired(c)
	if len(missing) != 5 {
		t.Errorf("expected 5 missing fields, got %d: %v", len(missing), missing)
	}

	// Should NOT include already-filled fields
	for _, m := range missing {
		if m == "Industry" || m == "Team size" || m == "Management pain points" {
			t.Errorf("field %q should not be in missing list", m)
		}
	}
}
