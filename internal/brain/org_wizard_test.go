package brain

import (
	"testing"
)

func TestParseWizardResponse_Continue(t *testing.T) {
	input := `{"status": "continue", "question": "What industry are you in?"}`

	resp, err := parseWizardResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "continue" {
		t.Errorf("expected status continue, got %q", resp.Status)
	}
	if resp.Question != "What industry are you in?" {
		t.Errorf("expected question about industry, got %q", resp.Question)
	}
	if resp.Profile != nil {
		t.Error("expected nil profile for continue status")
	}
}

func TestParseWizardResponse_Ready(t *testing.T) {
	input := `{"status": "ready", "profile": {"industry": "SaaS", "size": 15, "stage": "startup", "business_model": "B2B", "region": "SEA", "pain_points": ["hiring"]}}`

	resp, err := parseWizardResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ready" {
		t.Errorf("expected status ready, got %q", resp.Status)
	}
	if resp.Profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if resp.Profile.Industry != "SaaS" {
		t.Errorf("expected industry SaaS, got %q", resp.Profile.Industry)
	}
	if resp.Profile.Size != 15 {
		t.Errorf("expected size 15, got %d", resp.Profile.Size)
	}
	if resp.Profile.Stage != "startup" {
		t.Errorf("expected stage startup, got %q", resp.Profile.Stage)
	}
	if len(resp.Profile.PainPoints) != 1 || resp.Profile.PainPoints[0] != "hiring" {
		t.Errorf("expected pain points [hiring], got %v", resp.Profile.PainPoints)
	}
}

func TestParseWizardResponse_MarkdownWrapped(t *testing.T) {
	input := "```json\n{\"status\": \"continue\", \"question\": \"Tell me more\"}\n```"

	resp, err := parseWizardResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "continue" {
		t.Errorf("expected status continue, got %q", resp.Status)
	}
}

func TestParseWizardResponse_ExtraText(t *testing.T) {
	input := "Sure, let me ask: {\"status\": \"continue\", \"question\": \"How big is your team?\"} Hope that helps!"

	resp, err := parseWizardResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Question != "How big is your team?" {
		t.Errorf("unexpected question: %q", resp.Question)
	}
}

func TestParseWizardResponse_NoJSON(t *testing.T) {
	_, err := parseWizardResponse("no json here")
	if err == nil {
		t.Fatal("expected error for no JSON")
	}
}

func TestParseWizardResponse_InvalidJSON(t *testing.T) {
	_, err := parseWizardResponse("{invalid}")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuildWizardSystemPrompt(t *testing.T) {
	mentor := &MentorConfig{
		NameEn:     "Andy Grove",
		Company:    "Intel",
		Philosophy: "Only the paranoid survive",
		Strategy: Strategy{
			SystemPrompt: "Focus on output-oriented management.",
		},
	}

	prompt := buildWizardSystemPrompt(mentor)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !containsStr(prompt, "Andy Grove") {
		t.Error("prompt should contain mentor name")
	}
	if !containsStr(prompt, "Intel") {
		t.Error("prompt should contain company")
	}
	if !containsStr(prompt, "paranoid") {
		t.Error("prompt should contain philosophy")
	}
	if !containsStr(prompt, "output-oriented") {
		t.Error("prompt should contain system prompt content")
	}
	if !containsStr(prompt, "ready") {
		t.Error("prompt should contain 'ready' status instruction")
	}
	if !containsStr(prompt, "continue") {
		t.Error("prompt should contain 'continue' status instruction")
	}
}

func TestOrgWizard_Start_NilMentor(t *testing.T) {
	wizard := NewOrgWizard(nil)
	_, err := wizard.Start(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil mentor")
	}
}

func TestOrgWizard_ProcessAnswer_NilMentor(t *testing.T) {
	wizard := NewOrgWizard(nil)
	_, err := wizard.ProcessAnswer(nil, nil, nil, "answer")
	if err == nil {
		t.Fatal("expected error for nil mentor")
	}
}
