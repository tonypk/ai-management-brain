package onboarding

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// Planner generates a complete management plan from collected onboarding data.
type Planner struct {
	llm brain.LLMClient
}

// NewPlanner creates a new Planner with the given LLM client.
func NewPlanner(llm brain.LLMClient) *Planner {
	return &Planner{llm: llm}
}

const maxPlanRetries = 3

// GeneratePlan creates a complete management plan from collected onboarding data.
// It retries up to maxPlanRetries times if the LLM returns invalid JSON or a plan
// that fails validation, feeding the error back into the prompt for correction.
// LLM errors (network, auth, etc.) are returned immediately without retry.
func (p *Planner) GeneratePlan(ctx context.Context, data *CollectedData) (*ProposedPlan, error) {
	systemPrompt := buildPlanGenerationPrompt()
	userPrompt := fmt.Sprintf("Generate a management plan based on this company profile:\n%s", toJSON(data))

	for attempt := 0; attempt < maxPlanRetries; attempt++ {
		resp, err := p.llm.Chat(ctx, systemPrompt, userPrompt)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed: %w", err)
		}

		var plan ProposedPlan
		if err := json.Unmarshal([]byte(cleanJSON(resp)), &plan); err != nil {
			userPrompt = fmt.Sprintf("Your previous response was not valid JSON. Error: %s\nPlease try again with valid JSON only.", err)
			continue
		}

		if err := plan.Validate(); err != nil {
			userPrompt = fmt.Sprintf("Your plan was missing required fields: %s\nPlease include all required fields.", err)
			continue
		}

		return &plan, nil
	}

	return nil, fmt.Errorf("failed to generate valid plan after %d attempts", maxPlanRetries)
}

func buildPlanGenerationPrompt() string {
	return `You are a management systems architect. Given a company profile, generate a complete management plan as JSON.

The JSON must match this exact structure:
{
  "mentor": {"primary_id": "...", "secondary_id": "...", "blend_weight": 0.7, "reasoning": "..."},
  "board": [{"seat_type": "ceo|cfo|cmo|cto|chro|coo", "persona_id": "mentor_id", "reasoning": "..."}],
  "org_design": {
    "units": [{"ref_id": "...", "parent_ref_id": "", "name": "...", "unit_type": "department|team|squad", "head_role": "...", "responsibilities": "..."}],
    "reasoning": "..."
  },
  "policies": {
    "framework": "okr|kpi|scrum|mbo|bsc",
    "checkin_questions": ["..."],
    "tracking_focus": ["..."],
    "risk_rules": {"consecutive_misses": 3, "sentiment_drop_threshold": -0.3, "urgent_keywords": ["urgent"]},
    "cadence": {"daily_actions": [...], "weekly_actions": [...], "weekly_day": "friday", "monthly_actions": [...], "monthly_day": 1},
    "reasoning": "..."
  },
  "schedule": {"checkin": "0 9 * * 1-5", "chase": "30 17 * * 1-5", "summary": "0 19 * * 1-5", "briefing": "0 8 * * 1-5", "signal_scan": "*/30 9-18 * * 1-5", "timezone": "..."},
  "reasoning": "..."
}

Available mentor IDs: musk, inamori, ma, dalio, grove, ren, son, jobs, bezos, buffett, zhangyiming, leijun, caodewang, chushijian, meyer, trout

RESPOND WITH JSON ONLY. No markdown, no explanation.`
}
