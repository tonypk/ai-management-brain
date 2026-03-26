package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Ensure sqlc is used (for linter).
var _ = (*sqlc.Queries)(nil)

// ExecutionPlanner generates recommended actions based on company context,
// goal state, and execution signals.
type ExecutionPlanner struct {
	llm            LLMClient
	queries        *sqlc.Queries
	contextService *ContextService
}

// NewExecutionPlanner creates a new ExecutionPlanner.
func NewExecutionPlanner(llm LLMClient, queries *sqlc.Queries, cs *ContextService) *ExecutionPlanner {
	return &ExecutionPlanner{
		llm:            llm,
		queries:        queries,
		contextService: cs,
	}
}

// ExecutionPlan is the output of the planner.
type ExecutionPlan struct {
	Summary   string           `json:"summary"`
	Diagnosis PlanDiagnosis    `json:"diagnosis"`
	Actions   []PlannedAction  `json:"actions"`
}

// PlanDiagnosis captures what the planner identified.
type PlanDiagnosis struct {
	PrimaryIssue  string   `json:"primary_issue"`
	Owners        []string `json:"owners"`
	Signals       []string `json:"signals"`
	LinkedMetrics []string `json:"linked_metrics"`
	LinkedGoals   []string `json:"linked_goals"`
}

// PlannedAction is a recommended action.
type PlannedAction struct {
	Type               string `json:"type"` // create_task, follow_up, escalate, clarify, monitor
	Title              string `json:"title"`
	Owner              string `json:"owner"`
	Priority           string `json:"priority"` // critical, high, medium, low
	Reason             string `json:"reason"`
	DeadlineSuggestion string `json:"deadline_suggestion"`
}

const executionPlannerPrompt = `You are Execution Planner for Boss AI Agent.

Given company context, goal state, metric state, projects, and execution signals,
produce the best next actions for management execution.

INPUT: Structured context from company state.

OUTPUT JSON:
{
  "summary": "One sentence diagnosis",
  "diagnosis": {
    "primary_issue": "What's going wrong or needs attention",
    "owners": ["person names who own this"],
    "signals": ["relevant execution signals"],
    "linked_metrics": ["affected KPIs"],
    "linked_goals": ["affected OKRs"]
  },
  "actions": [
    {
      "type": "create_task | follow_up | escalate | clarify | monitor",
      "title": "Specific action description",
      "owner": "person name",
      "priority": "critical | high | medium | low",
      "reason": "Why this action, linked to data",
      "deadline_suggestion": "ISO date or relative like 'end of week'"
    }
  ]
}

PRIORITIZE actions that are:
1. High impact on off-track goals/metrics
2. Owner-clear (someone specific is responsible)
3. Workflow-compliant (respects company's approval rules and escalation paths)
4. Measurable (can verify completion)

NEVER recommend vague actions like "improve communication" or "try harder".
Every action must have a specific owner and measurable outcome.
Return ONLY valid JSON, no markdown code blocks.`

// Plan generates an execution plan for the given tenant.
func (ep *ExecutionPlanner) Plan(ctx context.Context, tenantID pgtype.UUID, focus string) (*ExecutionPlan, error) {
	// Get company context
	contextJSON, err := ep.contextService.FormatContextForPrompt(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get context: %w", err)
	}

	// Build user prompt
	userPrompt := fmt.Sprintf("Company context:\n%s", contextJSON)
	if focus != "" {
		userPrompt += fmt.Sprintf("\n\nFocus area: %s", focus)
	}

	// Call LLM
	resp, err := ep.llm.Chat(ctx, executionPlannerPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM execution planner: %w", err)
	}

	// Parse JSON
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var plan ExecutionPlan
	if err := json.Unmarshal([]byte(resp), &plan); err != nil {
		slog.Warn("failed to parse execution plan JSON",
			"response", resp, "error", err)
		return &ExecutionPlan{
			Summary: "Failed to generate structured plan. Raw response: " + resp,
		}, nil
	}

	slog.Info("generated execution plan",
		"actions_count", len(plan.Actions),
		"summary", plan.Summary,
	)

	return &plan, nil
}
