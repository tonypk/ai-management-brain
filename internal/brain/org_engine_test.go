package brain

import (
	"context"
	"encoding/json"
	"testing"
)

// mockAnthropicChat implements a mock for OrgEngine tests.
// OrgEngine uses *AnthropicClient directly (ChatLong), so we test via parseManagementPlan
// and through the exported types.

func TestParseManagementPlan_ValidJSON(t *testing.T) {
	input := `{
		"management_framework": "OKR",
		"org_design": {
			"philosophy": "flat and agile",
			"structure_type": "flat",
			"units": [
				{"name": "Engineering", "leader_type": "tech_lead", "leader_role": "Tech Lead", "size": 5, "kpis": ["velocity"]}
			],
			"support_roles": [
				{"title": "AI Assistant", "type": "ai", "scope": "daily standups"}
			]
		},
		"culture_principles": ["transparency", "ownership"],
		"policies": {"remote": "hybrid"},
		"kpi_system": [
			{"name": "Sprint Velocity", "target": "80 points", "frequency": "biweekly", "owner": "Tech Lead"}
		],
		"daily_questions": {
			"engineer": ["What did you ship?", "Any blockers?", "What's next?"]
		},
		"meeting_cadence": [
			{"name": "Daily Standup", "frequency": "daily", "duration": "15min", "attendees": "all", "purpose": "sync"}
		],
		"alert_rules": [
			{"condition": "miss_3_days", "action": "notify_lead", "message": "Check in on {name}"}
		],
		"reasoning": "Small team, flat structure works best"
	}`

	plan, err := parseManagementPlan(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Framework != "OKR" {
		t.Errorf("expected framework OKR, got %q", plan.Framework)
	}
	if plan.OrgDesign.StructureType != "flat" {
		t.Errorf("expected structure_type flat, got %q", plan.OrgDesign.StructureType)
	}
	if len(plan.OrgDesign.Units) != 1 {
		t.Errorf("expected 1 unit, got %d", len(plan.OrgDesign.Units))
	}
	if len(plan.OrgDesign.SupportRoles) != 1 {
		t.Errorf("expected 1 support role, got %d", len(plan.OrgDesign.SupportRoles))
	}
	if len(plan.CulturePrinciples) != 2 {
		t.Errorf("expected 2 culture principles, got %d", len(plan.CulturePrinciples))
	}
	if len(plan.KPISystem) != 1 {
		t.Errorf("expected 1 KPI, got %d", len(plan.KPISystem))
	}
	if questions, ok := plan.DailyQuestions["engineer"]; !ok || len(questions) != 3 {
		t.Errorf("expected 3 daily questions for engineer, got %v", plan.DailyQuestions)
	}
	if len(plan.MeetingCadence) != 1 {
		t.Errorf("expected 1 meeting, got %d", len(plan.MeetingCadence))
	}
	if len(plan.AlertRules) != 1 {
		t.Errorf("expected 1 alert rule, got %d", len(plan.AlertRules))
	}
	if plan.Reasoning == "" {
		t.Error("expected non-empty reasoning")
	}
}

func TestParseManagementPlan_MarkdownWrapped(t *testing.T) {
	input := "```json\n" + `{"management_framework":"KPI","org_design":{"philosophy":"p","structure_type":"hierarchy","units":[]},` +
		`"culture_principles":[],"policies":{},"kpi_system":[],"daily_questions":{},"meeting_cadence":[],"alert_rules":[],"reasoning":"r"}` + "\n```"

	plan, err := parseManagementPlan(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Framework != "KPI" {
		t.Errorf("expected framework KPI, got %q", plan.Framework)
	}
}

func TestParseManagementPlan_ExtraTextAround(t *testing.T) {
	input := "Here is my plan:\n\n" + `{"management_framework":"Scrum","org_design":{"philosophy":"p","structure_type":"flat","units":[]},` +
		`"culture_principles":[],"policies":{},"kpi_system":[],"daily_questions":{},"meeting_cadence":[],"alert_rules":[],"reasoning":"r"}` + "\n\nHope this helps!"

	plan, err := parseManagementPlan(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Framework != "Scrum" {
		t.Errorf("expected framework Scrum, got %q", plan.Framework)
	}
}

func TestParseManagementPlan_NoJSON(t *testing.T) {
	_, err := parseManagementPlan("no json here")
	if err == nil {
		t.Fatal("expected error for no JSON")
	}
}

func TestParseManagementPlan_InvalidJSON(t *testing.T) {
	_, err := parseManagementPlan("{invalid json}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuildOrgSystemPrompt(t *testing.T) {
	mentor := &MentorConfig{
		NameEn:     "Elon Musk",
		Company:    "SpaceX · Tesla",
		Philosophy: "第一性原理",
		Strategy: Strategy{
			SystemPrompt: "Think from first principles.",
		},
	}

	prompt := buildOrgSystemPrompt(mentor)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !contains(prompt, "Elon Musk") {
		t.Error("prompt should contain mentor name")
	}
	if !contains(prompt, "SpaceX") {
		t.Error("prompt should contain company")
	}
	if !contains(prompt, "第一性原理") {
		t.Error("prompt should contain philosophy")
	}
	if !contains(prompt, "first principles") {
		t.Error("prompt should contain system prompt")
	}
}

func TestBuildOrgUserPrompt(t *testing.T) {
	profile := CompanyProfile{
		Industry:      "SaaS",
		Size:          15,
		Stage:         "startup",
		BusinessModel: "B2B subscription",
		Region:        "Southeast Asia",
		PainPoints:    []string{"hiring", "retention"},
	}

	prompt := buildOrgUserPrompt(profile)
	if !contains(prompt, "SaaS") {
		t.Error("prompt should contain industry")
	}
	if !contains(prompt, "15") {
		t.Error("prompt should contain size")
	}
	if !contains(prompt, "startup") {
		t.Error("prompt should contain stage")
	}
	if !contains(prompt, "B2B subscription") {
		t.Error("prompt should contain business model")
	}
	if !contains(prompt, "Southeast Asia") {
		t.Error("prompt should contain region")
	}
	if !contains(prompt, "hiring") {
		t.Error("prompt should contain pain points")
	}
}

func TestBuildOrgUserPrompt_MinimalProfile(t *testing.T) {
	profile := CompanyProfile{
		Industry: "Tech",
		Size:     5,
		Stage:    "seed",
	}

	prompt := buildOrgUserPrompt(profile)
	if !contains(prompt, "Tech") {
		t.Error("prompt should contain industry")
	}
	// Should not contain optional fields
	if contains(prompt, "商业模式") {
		t.Error("prompt should not contain business model section for empty value")
	}
	if contains(prompt, "地区") {
		t.Error("prompt should not contain region section for empty value")
	}
}

func TestCompanyProfile_JSON_RoundTrip(t *testing.T) {
	profile := CompanyProfile{
		Industry:      "Fintech",
		Size:          50,
		Stage:         "growth",
		BusinessModel: "B2C",
		Region:        "Philippines",
		PainPoints:    []string{"compliance", "scaling"},
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded CompanyProfile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Industry != profile.Industry || decoded.Size != profile.Size {
		t.Errorf("roundtrip mismatch: got %+v", decoded)
	}
}

func TestManagementPlan_JSON_RoundTrip(t *testing.T) {
	plan := ManagementPlan{
		Framework: "OKR",
		OrgDesign: OrgDesign{
			Philosophy:    "flat",
			StructureType: "flat",
			Units:         []OrgUnit{{Name: "Eng", LeaderType: "lead", LeaderRole: "Tech Lead"}},
		},
		CulturePrinciples: []string{"ownership"},
		Policies:          map[string]interface{}{"remote": "yes"},
		KPISystem:         []KPIDefinition{{Name: "velocity", Target: "80", Frequency: "weekly", Owner: "lead"}},
		DailyQuestions:    map[string][]string{"all": {"What did you do?"}},
		MeetingCadence:    []MeetingRule{{Name: "standup", Frequency: "daily", Duration: "15m", Attendees: "all", Purpose: "sync"}},
		AlertRules:        []AlertRule{{Condition: "miss", Action: "notify", Message: "check"}},
		Reasoning:         "test",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ManagementPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Framework != plan.Framework {
		t.Errorf("framework mismatch: got %q", decoded.Framework)
	}
	if len(decoded.OrgDesign.Units) != 1 {
		t.Errorf("units mismatch: got %d", len(decoded.OrgDesign.Units))
	}
}

func TestOrgEngine_GeneratePlan_NilMentor(t *testing.T) {
	engine := NewOrgEngine(nil)
	_, err := engine.GeneratePlan(context.Background(), nil, CompanyProfile{})
	if err == nil {
		t.Fatal("expected error for nil mentor")
	}
}

func TestOrgEngine_AdjustPlan_NilPlan(t *testing.T) {
	engine := NewOrgEngine(nil)
	mentor := &MentorConfig{NameEn: "Test"}
	_, err := engine.AdjustPlan(context.Background(), mentor, nil, "feedback")
	if err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestOrgEngine_AdjustPlan_NilMentor(t *testing.T) {
	engine := NewOrgEngine(nil)
	plan := &ManagementPlan{Framework: "OKR"}
	_, err := engine.AdjustPlan(context.Background(), nil, plan, "feedback")
	if err == nil {
		t.Fatal("expected error for nil mentor")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
