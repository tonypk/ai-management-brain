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

// IncentiveEngine evaluates incentive scores for employees based on
// active rules, execution data, and AI analysis.
type IncentiveEngine struct {
	llm            LLMClient
	queries        *sqlc.Queries
	contextService *ContextService
}

// NewIncentiveEngine creates a new IncentiveEngine.
func NewIncentiveEngine(llm LLMClient, queries *sqlc.Queries, cs *ContextService) *IncentiveEngine {
	return &IncentiveEngine{
		llm:            llm,
		queries:        queries,
		contextService: cs,
	}
}

// IncentiveResult is the LLM evaluation result for one person + rule.
type IncentiveResult struct {
	PersonID              string                 `json:"person_id"`
	Period                string                 `json:"period"`
	Eligible              bool                   `json:"eligible"`
	Score                 float64                `json:"score"`
	ScoreBreakdown        map[string]interface{} `json:"score_breakdown"`
	AttributionConfidence float64                `json:"attribution_confidence"`
	PayoutWeight          float64                `json:"payout_weight"`
	Highlights            []string               `json:"highlights"`
	Concerns              []string               `json:"concerns"`
	NeedsReview           bool                   `json:"needs_review"`
}

const incentiveEvaluatorPrompt = `You are Incentive Evaluator for Boss AI Agent.

Evaluate incentive consequences based on:
- Active incentive rules (reward model, attribution rules, scoring formula)
- Employee's execution data (tasks completed, metric contributions, signal history)
- Goal attribution (which goals did this person directly impact)
- Communication quality (proactive updates, acknowledgments, blocks reported early)

RULES:
- Do not reward outcomes that are not attributable by the company's rules
- Do not assign penalties outside defined policy
- When attribution is ambiguous (multiple people contributed), mark for human review
- Factor in execution quality, not just outcomes (consistent delivery > last-minute heroics)

OUTPUT JSON:
{
  "person_id": "...",
  "period": "2026-03",
  "eligible": true,
  "score": 85.5,
  "score_breakdown": {
    "goal_progress": 40,
    "execution_quality": 25,
    "communication": 15,
    "initiative": 5
  },
  "attribution_confidence": 0.82,
  "payout_weight": 0.85,
  "highlights": ["Completed payment gateway migration ahead of schedule"],
  "concerns": ["Test coverage below 60% on new modules"],
  "needs_review": false
}

Return ONLY valid JSON, no markdown code blocks.`

// Calculate evaluates incentive scores for a specific employee and period.
func (ie *IncentiveEngine) Calculate(
	ctx context.Context,
	tenantID pgtype.UUID,
	period string,
	personID pgtype.UUID,
	personName string,
) ([]sqlc.IncentiveScore, error) {
	// Get active incentive rules
	rules, err := ie.queries.ListIncentiveRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list incentive rules: %w", err)
	}

	if len(rules) == 0 {
		return nil, nil
	}

	// Gather context for the person
	contextJSON, err := ie.contextService.FormatContextForPrompt(ctx, tenantID)
	if err != nil {
		slog.Warn("incentive_engine: failed to get context", "error", err)
		contextJSON = "{}"
	}

	// Get person's execution signals
	signals, err := ie.queries.GetSignalsBySubject(ctx, sqlc.GetSignalsBySubjectParams{
		SubjectType: "person",
		SubjectID:   personID,
		Limit:       50,
	})
	if err != nil {
		slog.Warn("incentive_engine: failed to get signals", "error", err)
	}

	// Get person's tasks
	tasks, err := ie.queries.ListTasksByOwner(ctx, personID)
	if err != nil {
		slog.Warn("incentive_engine: failed to get tasks", "error", err)
	}

	var results []sqlc.IncentiveScore

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		// Build user prompt with all context
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Employee: %s\n", personName))
		sb.WriteString(fmt.Sprintf("Period: %s\n\n", period))

		sb.WriteString(fmt.Sprintf("Incentive Rule: %s\n", rule.Name))
		sb.WriteString(fmt.Sprintf("Reward Model: %s\n", rule.RewardModel))
		sb.WriteString(fmt.Sprintf("Payout Cycle: %s\n", rule.PayoutCycle))
		if rule.AttributionRules != nil {
			sb.WriteString(fmt.Sprintf("Attribution Rules: %s\n", string(rule.AttributionRules)))
		}
		if rule.ScoringFormula != nil {
			sb.WriteString(fmt.Sprintf("Scoring Formula: %s\n", string(rule.ScoringFormula)))
		}

		sb.WriteString(fmt.Sprintf("\nCompany Context:\n%s\n", contextJSON))

		if len(signals) > 0 {
			sb.WriteString("\nExecution Signals:\n")
			for _, s := range signals {
				sb.WriteString(fmt.Sprintf("- %s (score: %v, reasons: %s)\n",
					s.SignalType, s.Score, string(s.Reasons)))
			}
		}

		if len(tasks) > 0 {
			sb.WriteString(fmt.Sprintf("\nTasks: %d total\n", len(tasks)))
			done := 0
			for _, t := range tasks {
				if t.Status == "done" {
					done++
				}
			}
			sb.WriteString(fmt.Sprintf("Completed: %d\n", done))
		}

		// Call LLM
		resp, err := ie.llm.Chat(ctx, incentiveEvaluatorPrompt, sb.String())
		if err != nil {
			slog.Error("incentive_engine: LLM evaluation failed",
				"rule", rule.Name, "person", personName, "error", err)
			continue
		}

		// Parse response
		resp = strings.TrimSpace(resp)
		if strings.HasPrefix(resp, "```") {
			lines := strings.Split(resp, "\n")
			if len(lines) > 2 {
				resp = strings.Join(lines[1:len(lines)-1], "\n")
			}
		}

		var result IncentiveResult
		if err := json.Unmarshal([]byte(resp), &result); err != nil {
			slog.Warn("incentive_engine: failed to parse result JSON",
				"response", resp, "error", err)
			continue
		}

		// Store the score
		breakdown, _ := json.Marshal(result.ScoreBreakdown)
		var score pgtype.Numeric
		_ = score.Scan(fmt.Sprintf("%.2f", result.Score))
		var payoutWeight pgtype.Numeric
		_ = payoutWeight.Scan(fmt.Sprintf("%.2f", result.PayoutWeight))
		var attrConf pgtype.Numeric
		_ = attrConf.Scan(fmt.Sprintf("%.2f", result.AttributionConfidence))

		status := "calculated"
		if result.NeedsReview {
			status = "needs_review"
		}

		stored, err := ie.queries.CreateIncentiveScore(ctx, sqlc.CreateIncentiveScoreParams{
			TenantID:              tenantID,
			RuleID:                rule.ID,
			PersonID:              personID,
			Period:                period,
			Score:                 score,
			ScoreBreakdown:        breakdown,
			PayoutWeight:          payoutWeight,
			AttributionConfidence: attrConf,
			Status:                status,
		})
		if err != nil {
			slog.Error("incentive_engine: store score failed",
				"rule", rule.Name, "person", personName, "error", err)
			continue
		}
		results = append(results, stored)
	}

	slog.Info("incentive_engine: calculated scores",
		"person", personName,
		"period", period,
		"scores_count", len(results),
	)

	return results, nil
}

// Preview evaluates all active employees for a period without storing (dry-run).
func (ie *IncentiveEngine) Preview(
	ctx context.Context,
	tenantID pgtype.UUID,
	period string,
) ([]IncentiveResult, error) {
	employees, err := ie.queries.ListActiveEmployees(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}

	rules, err := ie.queries.ListIncentiveRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}

	if len(rules) == 0 || len(employees) == 0 {
		return nil, nil
	}

	contextJSON, _ := ie.contextService.FormatContextForPrompt(ctx, tenantID)

	var previews []IncentiveResult

	for _, emp := range employees {
		for _, rule := range rules {
			if !rule.IsActive {
				continue
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Employee: %s\nPeriod: %s\n", emp.Name, period))
			sb.WriteString(fmt.Sprintf("Rule: %s (%s, %s)\n", rule.Name, rule.RewardModel, rule.PayoutCycle))
			sb.WriteString(fmt.Sprintf("Context:\n%s\n", contextJSON))

			resp, err := ie.llm.Chat(ctx, incentiveEvaluatorPrompt, sb.String())
			if err != nil {
				slog.Error("incentive_engine: preview LLM failed",
					"employee", emp.Name, "error", err)
				continue
			}

			resp = strings.TrimSpace(resp)
			if strings.HasPrefix(resp, "```") {
				lines := strings.Split(resp, "\n")
				if len(lines) > 2 {
					resp = strings.Join(lines[1:len(lines)-1], "\n")
				}
			}

			var result IncentiveResult
			if err := json.Unmarshal([]byte(resp), &result); err != nil {
				continue
			}
			result.PersonID = formatUUID(emp.ID)
			result.Period = period
			previews = append(previews, result)
		}
	}

	return previews, nil
}

// formatUUID converts a pgtype.UUID to string.
func formatUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
