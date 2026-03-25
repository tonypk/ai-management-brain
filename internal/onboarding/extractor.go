package onboarding

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// Extractor extracts structured info from onboarding dialogue using a lightweight LLM call.
type Extractor struct {
	llm brain.LLMClient // single-turn, Haiku-class
}

// NewExtractor creates a new Extractor with the given LLM client.
func NewExtractor(llm brain.LLMClient) *Extractor {
	return &Extractor{llm: llm}
}

// ExtractInfo takes the current collected data and a new user message,
// returns updated collected data with any new info merged in.
func (e *Extractor) ExtractInfo(ctx context.Context, current *CollectedData, userMessage string) (*CollectedData, error) {
	prompt := BuildExtractionPrompt(current, userMessage)
	resp, err := e.llm.Chat(ctx, "You are a JSON extraction assistant. Return ONLY valid JSON.", prompt)
	if err != nil {
		slog.Warn("extraction LLM call failed", "error", err)
		return current, nil // non-fatal: continue with what we have
	}

	var delta CollectedData
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &delta); err != nil {
		slog.Warn("extraction JSON parse failed", "error", err, "response", resp)
		return current, nil
	}

	return mergeCollectedData(current, &delta), nil
}

// mergeCollectedData merges new non-zero fields from delta into base (immutable — returns new copy).
func mergeCollectedData(base, delta *CollectedData) *CollectedData {
	result := *base // copy
	if delta.Industry != "" {
		result.Industry = delta.Industry
	}
	if delta.CompanyStage != "" {
		result.CompanyStage = delta.CompanyStage
	}
	if delta.BusinessModel != "" {
		result.BusinessModel = delta.BusinessModel
	}
	if delta.TeamSize > 0 {
		result.TeamSize = delta.TeamSize
	}
	if delta.OrgStructure != "" {
		result.OrgStructure = delta.OrgStructure
	}
	if delta.CurrentProjects != "" {
		result.CurrentProjects = delta.CurrentProjects
	}
	if len(delta.PainPoints) > 0 {
		result.PainPoints = delta.PainPoints
	}
	if len(delta.CommTools) > 0 {
		result.CommTools = delta.CommTools
	}
	if delta.CulturePrefs != "" {
		result.CulturePrefs = delta.CulturePrefs
	}
	if delta.GoalFramework != "" {
		result.GoalFramework = delta.GoalFramework
	}
	return &result
}
