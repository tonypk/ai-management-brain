package worldmodel_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/worldmodel"
)

func TestParseExtractionResponse(t *testing.T) {
	raw := `{
		"skills": [{"name": "React", "proficiency": "high"}],
		"relationships": [{"colleague": "Alice", "type": "collaborates", "context": "frontend"}],
		"blockers": [{"category": "cross_team", "description": "waiting for backend API", "resolved": false}],
		"growth_events": [{"type": "skill_upgrade", "description": "First solo backend API"}]
	}`

	result, err := worldmodel.ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("ParseExtractionResponse: %v", err)
	}
	if len(result.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "React" {
		t.Errorf("expected skill name React, got %s", result.Skills[0].Name)
	}
	if result.Skills[0].Proficiency != "high" {
		t.Errorf("expected proficiency high, got %s", result.Skills[0].Proficiency)
	}
	if len(result.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(result.Relationships))
	}
	if len(result.Blockers) != 1 {
		t.Errorf("expected 1 blocker, got %d", len(result.Blockers))
	}
	if len(result.GrowthEvents) != 1 {
		t.Errorf("expected 1 growth event, got %d", len(result.GrowthEvents))
	}
}

func TestParseExtractionResponse_MarkdownCodeBlock(t *testing.T) {
	raw := "```json\n{\"skills\": [], \"relationships\": [], \"blockers\": [], \"growth_events\": []}\n```"
	result, err := worldmodel.ParseExtractionResponse(raw)
	if err != nil {
		t.Fatalf("ParseExtractionResponse with markdown: %v", err)
	}
	if len(result.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(result.Skills))
	}
}

func TestParseExtractionResponse_InvalidJSON(t *testing.T) {
	_, err := worldmodel.ParseExtractionResponse("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBuildExtractionPrompt(t *testing.T) {
	existing := &worldmodel.EmployeeWorldModel{
		Skills: []worldmodel.SkillEntry{{Name: "Go", Proficiency: "medium"}},
	}
	prompt := worldmodel.BuildExtractionPrompt("What did you do today? I worked on the React dashboard with Bob.", existing)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
}
