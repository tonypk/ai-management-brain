package worldmodel

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// ---------------------------------------------------------------------------
// Trigger 1: Blocker Escalation
// ---------------------------------------------------------------------------

// BlockerEscalationInput holds data for the blocker escalation trigger.
type BlockerEscalationInput struct {
	EmployeeID      string
	EmployeeName    string
	Category        string
	Description     string
	RecurrenceCount int
	FirstSeenAt     string
}

// EvalBlockerEscalation returns a recommendation if the blocker has recurred >= 3 times.
func EvalBlockerEscalation(input BlockerEscalationInput) *brain.RecommendationInput {
	if input.RecurrenceCount < 3 {
		return nil
	}

	actions, err := json.Marshal([]map[string]any{
		{"type": "flag_risk", "params": map[string]any{
			"risk_description": fmt.Sprintf("%s blocker recurring %dx for %s", input.Category, input.RecurrenceCount, input.EmployeeName),
		}, "label": "Flag risk"},
		{"type": "schedule_meeting", "params": map[string]any{
			"employee_id": input.EmployeeID, "meeting_type": "one_on_one",
			"notes": fmt.Sprintf("Discuss recurring %s blocker (%dx)", input.Category, input.RecurrenceCount),
		}, "label": "Schedule 1:1"},
	})
	if err != nil {
		slog.Error("EvalBlockerEscalation: marshal actions failed", "error", err)
	}

	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": input.EmployeeName, "issue": "blocker_escalation"}},
		"world_model_evidence": map[string]any{
			"type":             "blocker_escalation",
			"category":         input.Category,
			"description":      input.Description,
			"recurrence_count": input.RecurrenceCount,
			"first_seen_at":    input.FirstSeenAt,
		},
	})
	if err != nil {
		slog.Error("EvalBlockerEscalation: marshal evidence failed", "error", err)
	}

	entityType := "employee"
	return &brain.RecommendationInput{
		Category:         "people",
		Priority:         "high",
		Title:            fmt.Sprintf("%s's %s blocker keeps recurring (%dx)", input.EmployeeName, input.Category, input.RecurrenceCount),
		Description:      fmt.Sprintf("%s has had a '%s' blocker recurring %d times since %s: %s. Consider scheduling a meeting to address the root cause.", input.EmployeeName, input.Category, input.RecurrenceCount, input.FirstSeenAt, input.Description),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &input.EmployeeID,
	}
}

// ---------------------------------------------------------------------------
// Trigger 2: Skill Match
// ---------------------------------------------------------------------------

// SkillMatchInput holds data for the skill-blocker match trigger.
type SkillMatchInput struct {
	BlockedEmployeeID   string
	BlockedEmployeeName string
	BlockerCategory     string
	HelperEmployeeID    string
	HelperEmployeeName  string
	HelperSkillName     string
	HelperConfidence    float64
}

// EvalSkillMatch returns a recommendation to pair a blocked employee with a helper.
func EvalSkillMatch(input SkillMatchInput) *brain.RecommendationInput {
	actions, err := json.Marshal([]map[string]any{
		{"type": "send_message", "params": map[string]any{
			"employee_id": input.HelperEmployeeID,
			"message":     fmt.Sprintf("Hi %s, %s is working through a %s challenge. Would you be open to a quick pairing session to help?", input.HelperEmployeeName, input.BlockedEmployeeName, input.BlockerCategory),
		}, "label": fmt.Sprintf("Notify %s", input.HelperEmployeeName)},
		{"type": "send_message", "params": map[string]any{
			"employee_id": input.BlockedEmployeeID,
			"message":     fmt.Sprintf("Hi %s, %s has experience with %s and might be able to help with your current blocker. Consider reaching out!", input.BlockedEmployeeName, input.HelperEmployeeName, input.HelperSkillName),
		}, "label": fmt.Sprintf("Notify %s", input.BlockedEmployeeName)},
	})
	if err != nil {
		slog.Error("EvalSkillMatch: marshal actions failed", "error", err)
	}

	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{
			{"name": input.BlockedEmployeeName, "issue": "blocked_on_" + input.BlockerCategory},
			{"name": input.HelperEmployeeName, "issue": "has_matching_skill"},
		},
		"world_model_evidence": map[string]any{
			"type":              "skill_match",
			"blocker_category":  input.BlockerCategory,
			"helper_skill":      input.HelperSkillName,
			"helper_confidence": input.HelperConfidence,
		},
	})
	if err != nil {
		slog.Error("EvalSkillMatch: marshal evidence failed", "error", err)
	}

	return &brain.RecommendationInput{
		Category:         "people",
		Priority:         "medium",
		Title:            fmt.Sprintf("Pair %s with %s to solve %s blocker", input.BlockedEmployeeName, input.HelperEmployeeName, input.BlockerCategory),
		Description:      fmt.Sprintf("%s has a %s blocker. %s has %s expertise (confidence %d%%). A pairing session could resolve this.", input.BlockedEmployeeName, input.BlockerCategory, input.HelperEmployeeName, input.HelperSkillName, int(input.HelperConfidence*100)),
		SuggestedActions: actions,
		Evidence:         evidence,
	}
}

// ---------------------------------------------------------------------------
// Trigger 3: Compound Risk (sentiment decline + active blockers)
// ---------------------------------------------------------------------------

// CompoundRiskInput holds data for the compound risk trigger.
type CompoundRiskInput struct {
	EmployeeID     string
	EmployeeName   string
	SentimentTrend []string // most recent first: ["negative", "negative", "neutral"]
	ActiveBlockers []string // blocker categories
}

// sentimentScore maps sentiment text to a numeric score.
func sentimentScore(s string) int {
	switch strings.ToLower(s) {
	case "positive":
		return 3
	case "neutral":
		return 2
	case "negative":
		return 1
	default:
		return 2
	}
}

// isDecliningSentiment checks if sentiment has been declining over the trend window.
// Requires at least 3 data points. trend[0] is most recent.
// Returns true if scores are non-increasing AND most recent < oldest (actual decline).
func isDecliningSentiment(trend []string) bool {
	if len(trend) < 3 {
		return false
	}
	scores := make([]int, len(trend))
	for i, s := range trend {
		scores[i] = sentimentScore(s)
	}
	// Most recent should be lowest (declining)
	if scores[0] >= scores[len(scores)-1] {
		return false
	}
	// Each day should be <= the day before (non-increasing from recent to old)
	for i := 0; i < len(scores)-1; i++ {
		if scores[i] > scores[i+1] {
			return false
		}
	}
	return true
}

// EvalCompoundRisk returns a recommendation if sentiment is declining AND employee has active blockers.
func EvalCompoundRisk(input CompoundRiskInput) *brain.RecommendationInput {
	if len(input.ActiveBlockers) == 0 {
		return nil
	}
	if !isDecliningSentiment(input.SentimentTrend) {
		return nil
	}

	actions, err := json.Marshal([]map[string]any{
		{"type": "schedule_meeting", "params": map[string]any{
			"employee_id": input.EmployeeID, "meeting_type": "one_on_one",
			"notes": "Wellness check: declining sentiment with active blockers",
		}, "label": "Schedule 1:1"},
		{"type": "send_message", "params": map[string]any{
			"employee_id": input.EmployeeID,
			"message":     fmt.Sprintf("Hi %s, I wanted to check in — how are things going? Is there anything I can help unblock?", input.EmployeeName),
		}, "label": "Send care message"},
	})
	if err != nil {
		slog.Error("EvalCompoundRisk: marshal actions failed", "error", err)
	}

	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": input.EmployeeName, "issue": "compound_risk"}},
		"world_model_evidence": map[string]any{
			"type":            "compound_risk",
			"sentiment_trend": input.SentimentTrend,
			"active_blockers": input.ActiveBlockers,
			"blocker_count":   len(input.ActiveBlockers),
		},
	})
	if err != nil {
		slog.Error("EvalCompoundRisk: marshal evidence failed", "error", err)
	}

	entityType := "employee"
	return &brain.RecommendationInput{
		Category:         "people",
		Priority:         "high",
		Title:            fmt.Sprintf("%s: declining sentiment with %d unresolved blockers", input.EmployeeName, len(input.ActiveBlockers)),
		Description:      fmt.Sprintf("%s's sentiment has been declining (trend: %s) while having %d active blockers (%s). Recommend a proactive 1:1.", input.EmployeeName, strings.Join(input.SentimentTrend, " → "), len(input.ActiveBlockers), strings.Join(input.ActiveBlockers, ", ")),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &input.EmployeeID,
	}
}
