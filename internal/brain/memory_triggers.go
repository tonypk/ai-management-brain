package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/memory"
)

// MemoryPatternEvaluator checks employee memory patterns and generates recommendations.
type MemoryPatternEvaluator struct {
	memStore *memory.MemoryStore
}

// NewMemoryPatternEvaluator creates a new evaluator.
func NewMemoryPatternEvaluator(memStore *memory.MemoryStore) *MemoryPatternEvaluator {
	return &MemoryPatternEvaluator{memStore: memStore}
}

// EvaluateAfterExtraction checks memory patterns for an employee and returns recommendations.
// Called after ExtractFromReport/ExtractFromChase to detect emerging patterns.
func (e *MemoryPatternEvaluator) EvaluateAfterExtraction(
	ctx context.Context,
	tenantID, employeeID, employeeName string,
) []RecommendationInput {
	if e.memStore == nil {
		return nil
	}

	var results []RecommendationInput

	// Get recent short-term memories (last 14 days)
	memories, err := e.memStore.List(ctx, tenantID, "employee_insight", "short_term", employeeID, 20, 0)
	if err != nil || len(memories) < 3 {
		return nil // Need at least 3 memories to detect patterns
	}

	// Check stress pattern
	if rec := e.checkStressPattern(memories, employeeID, employeeName); rec != nil {
		results = append(results, *rec)
	}

	// Check repeated blocker
	if rec := e.checkRepeatedBlocker(memories, employeeID, employeeName); rec != nil {
		results = append(results, *rec)
	}

	// Check growth signal
	if rec := e.checkGrowthSignal(memories, employeeID, employeeName); rec != nil {
		results = append(results, *rec)
	}

	// Check engagement drop
	if rec := e.checkEngagementDrop(ctx, tenantID, employeeID, employeeName); rec != nil {
		results = append(results, *rec)
	}

	return results
}

// stressKeywords are words that indicate stress/pressure/burnout.
var stressKeywords = []string{
	"stress", "stressed", "pressure", "burnout", "burnt", "overwhelmed",
	"overwork", "overtime", "exhausted", "tired", "frustrated", "anxiety",
	"压力", "加班", "疲惫", "焦虑", "劳累", "累",
}

// checkStressPattern detects 3+ stress-related memories in recent history.
func (e *MemoryPatternEvaluator) checkStressPattern(
	memories []memory.Memory, employeeID, employeeName string,
) *RecommendationInput {
	stressCount := 0
	var stressEvidence []map[string]any

	for _, m := range memories {
		lower := strings.ToLower(m.Content)
		for _, kw := range stressKeywords {
			if strings.Contains(lower, kw) {
				stressCount++
				stressEvidence = append(stressEvidence, map[string]any{
					"date":       m.CreatedAt.Format("2006-01-02"),
					"content":    m.Content,
					"importance": m.Importance,
				})
				break
			}
		}
	}

	if stressCount < 3 {
		return nil
	}

	actions, _ := json.Marshal([]map[string]any{
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": employeeID, "meeting_type": "one_on_one", "notes": "Discuss workload and wellbeing"}, "label": "Schedule 1:1"},
		{"type": "send_message", "params": map[string]any{"employee_id": employeeID, "message": fmt.Sprintf("Hi %s, I've noticed things have been intense lately. Want to chat about how I can help?", employeeName)}, "label": "Send supportive message"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees":       []map[string]any{{"name": employeeName, "issue": fmt.Sprintf("stress_pattern_%d_mentions", stressCount)}},
		"memory_evidence": stressEvidence,
	})

	entityType := "employee"
	return &RecommendationInput{
		Category:         "people",
		Priority:         "high",
		Title:            fmt.Sprintf("Check on %s — recurring stress signals", employeeName),
		Description:      fmt.Sprintf("%s has mentioned stress-related topics %d times in recent check-ins. This pattern suggests workload or wellbeing concerns that deserve attention.", employeeName, stressCount),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &employeeID,
	}
}

// blockerKeywords are words that indicate blockers.
var blockerKeywords = []string{
	"blocked", "blocker", "stuck", "waiting", "depends on", "dependency",
	"can't proceed", "cannot proceed", "holding up", "delayed",
	"阻碍", "卡住", "等待", "依赖", "延迟",
}

// checkRepeatedBlocker detects same-theme blocker in 3+ memories.
func (e *MemoryPatternEvaluator) checkRepeatedBlocker(
	memories []memory.Memory, employeeID, employeeName string,
) *RecommendationInput {
	blockerMentions := 0
	var blockerEvidence []map[string]any

	for _, m := range memories {
		lower := strings.ToLower(m.Content)
		for _, kw := range blockerKeywords {
			if strings.Contains(lower, kw) {
				blockerMentions++
				blockerEvidence = append(blockerEvidence, map[string]any{
					"date":       m.CreatedAt.Format("2006-01-02"),
					"content":    m.Content,
					"importance": m.Importance,
				})
				break
			}
		}
	}

	if blockerMentions < 3 {
		return nil
	}

	actions, _ := json.Marshal([]map[string]any{
		{"type": "create_task", "params": map[string]any{"title": fmt.Sprintf("Resolve recurring blocker for %s", employeeName), "description": "Multiple check-ins mention the same blocking issue"}, "label": "Create resolution task"},
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": employeeID, "meeting_type": "one_on_one", "notes": "Discuss persistent blockers"}, "label": "Schedule 1:1"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees":       []map[string]any{{"name": employeeName, "issue": fmt.Sprintf("repeated_blocker_%d_mentions", blockerMentions)}},
		"memory_evidence": blockerEvidence,
	})

	entityType := "employee"
	return &RecommendationInput{
		Category:         "project",
		Priority:         "high",
		Title:            fmt.Sprintf("Persistent blocker for %s — mentioned %d times", employeeName, blockerMentions),
		Description:      fmt.Sprintf("%s has mentioned blockers or delays %d times recently. This recurring pattern suggests a systemic issue that needs escalation.", employeeName, blockerMentions),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &employeeID,
	}
}

// growthKeywords are words that indicate positive growth.
var growthKeywords = []string{
	"completed", "shipped", "achieved", "exceeded", "improved",
	"initiative", "proactive", "volunteered", "led", "mentored",
	"完成", "超越", "提升", "主动", "改进",
}

// checkGrowthSignal detects 3+ positive memories.
func (e *MemoryPatternEvaluator) checkGrowthSignal(
	memories []memory.Memory, employeeID, employeeName string,
) *RecommendationInput {
	growthCount := 0
	var growthEvidence []map[string]any

	for _, m := range memories {
		lower := strings.ToLower(m.Content)
		for _, kw := range growthKeywords {
			if strings.Contains(lower, kw) {
				growthCount++
				growthEvidence = append(growthEvidence, map[string]any{
					"date":       m.CreatedAt.Format("2006-01-02"),
					"content":    m.Content,
					"importance": m.Importance,
				})
				break
			}
		}
	}

	if growthCount < 3 {
		return nil
	}

	actions, _ := json.Marshal([]map[string]any{
		{"type": "public_recognition", "params": map[string]any{"employee_id": employeeID, "message": fmt.Sprintf("Shoutout to %s for consistent excellence! Keep up the great work.", employeeName)}, "label": "Public recognition"},
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": employeeID, "meeting_type": "one_on_one", "notes": "Discuss career growth and next steps"}, "label": "Discuss growth opportunities"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees":       []map[string]any{{"name": employeeName, "issue": fmt.Sprintf("growth_signal_%d_positives", growthCount)}},
		"memory_evidence": growthEvidence,
	})

	entityType := "employee"
	return &RecommendationInput{
		Category:         "people",
		Priority:         "medium",
		Title:            fmt.Sprintf("Recognize %s — consistent positive pattern", employeeName),
		Description:      fmt.Sprintf("%s has shown %d positive signals recently (achievements, initiative, growth). Consider recognition and career development discussion.", employeeName, growthCount),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &employeeID,
	}
}

// checkEngagementDrop detects declining memory frequency (proxy for engagement).
func (e *MemoryPatternEvaluator) checkEngagementDrop(
	ctx context.Context, tenantID, employeeID, employeeName string,
) *RecommendationInput {
	// Compare recent 7 days vs previous 7 days memory count
	allMemories, err := e.memStore.List(ctx, tenantID, "employee_insight", "", employeeID, 50, 0)
	if err != nil || len(allMemories) < 5 {
		return nil // Need sufficient history
	}

	// Count memories in two periods
	now := allMemories[0].CreatedAt // Most recent memory
	recentCount := 0
	previousCount := 0
	for _, m := range allMemories {
		daysSince := now.Sub(m.CreatedAt).Hours() / 24
		if daysSince <= 7 {
			recentCount++
		} else if daysSince <= 14 {
			previousCount++
		}
	}

	// Engagement drop: previous had 3+ memories but recent has 0-1
	if previousCount < 3 || recentCount > 1 {
		return nil
	}

	actions, _ := json.Marshal([]map[string]any{
		{"type": "send_message", "params": map[string]any{"employee_id": employeeID, "message": fmt.Sprintf("Hi %s, just checking in — how are things going?", employeeName)}, "label": "Send check-in message"},
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": employeeID, "meeting_type": "one_on_one"}, "label": "Schedule 1:1"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": employeeName, "issue": fmt.Sprintf("engagement_drop_from_%d_to_%d", previousCount, recentCount)}},
	})

	entityType := "employee"
	return &RecommendationInput{
		Category:         "people",
		Priority:         "medium",
		Title:            fmt.Sprintf("Engagement drop for %s", employeeName),
		Description:      fmt.Sprintf("%s's check-in engagement dropped significantly (from %d to %d insights in the past week). Consider proactive outreach.", employeeName, previousCount, recentCount),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &employeeID,
	}
}
