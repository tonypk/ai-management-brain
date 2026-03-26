package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	sqlc "github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Recommender generates AI management recommendations via daily scan and realtime triggers.
// Uses *AnthropicClient directly (not LLMClient interface) to access ChatLong() for daily scan.
type Recommender struct {
	llm            *AnthropicClient
	queries        *sqlc.Queries
	contextService *ContextService
}

// RecommendationInput holds data for a single recommendation to be created.
type RecommendationInput struct {
	Category         string          `json:"category"`
	Priority         string          `json:"priority"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	SuggestedActions json.RawMessage `json:"suggested_actions"`
	Evidence         json.RawMessage `json:"evidence"`
	TargetEntityType *string         `json:"target_entity_type,omitempty"`
	TargetEntityID   *string         `json:"target_entity_id,omitempty"`
}

// NewRecommender creates a new Recommender.
func NewRecommender(llm *AnthropicClient, queries *sqlc.Queries, cs *ContextService) *Recommender {
	return &Recommender{llm: llm, queries: queries, contextService: cs}
}

// ---------------------------------------------------------------------------
// 4 template generators (no LLM cost)
// ---------------------------------------------------------------------------

func (r *Recommender) templateConsecutiveMiss(emp sqlc.Employee, days int64) RecommendationInput {
	priority := "high"
	if days >= 5 {
		priority = "critical"
	}
	empID := uuidToString(emp.ID)
	actions, err := json.Marshal([]map[string]any{
		{"type": "send_message", "params": map[string]any{"employee_id": empID, "message": fmt.Sprintf("Hi %s, how are you doing? Is there anything I can help with?", emp.Name)}, "label": "Send care message"},
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": empID, "meeting_type": "one_on_one"}, "label": "Schedule 1:1"},
	})
	if err != nil {
		slog.Error("templateConsecutiveMiss: marshal actions failed", "error", err)
	}
	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": emp.Name, "issue": fmt.Sprintf("consecutive_miss_%dd", days)}},
	})
	if err != nil {
		slog.Error("templateConsecutiveMiss: marshal evidence failed", "error", err)
	}
	entityType := "employee"
	return RecommendationInput{
		Category:         "people",
		Priority:         priority,
		Title:            fmt.Sprintf("Follow up with %s (%d days no check-in)", emp.Name, days),
		Description:      fmt.Sprintf("%s has not submitted a check-in for %d consecutive days. Consider reaching out to understand the situation.", emp.Name, days),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &empID,
	}
}

func (r *Recommender) templateSentimentDrop(emp sqlc.Employee, trend string) RecommendationInput {
	empID := uuidToString(emp.ID)
	actions, err := json.Marshal([]map[string]any{
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": empID, "meeting_type": "one_on_one", "notes": "Discuss recent sentiment trend"}, "label": "Schedule 1:1"},
		{"type": "send_message", "params": map[string]any{"employee_id": empID, "message": fmt.Sprintf("Hi %s, I noticed things have been tough lately. Want to chat?", emp.Name)}, "label": "Send supportive message"},
	})
	if err != nil {
		slog.Error("templateSentimentDrop: marshal actions failed", "error", err)
	}
	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": emp.Name, "issue": "sentiment_drop_3d"}},
	})
	if err != nil {
		slog.Error("templateSentimentDrop: marshal evidence failed", "error", err)
	}
	entityType := "employee"
	return RecommendationInput{
		Category:         "people",
		Priority:         "high",
		Title:            fmt.Sprintf("Check on %s — declining sentiment", emp.Name),
		Description:      fmt.Sprintf("%s has shown 3 consecutive days of negative sentiment (%s). A proactive 1:1 may help.", emp.Name, trend),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &empID,
	}
}

func (r *Recommender) templatePerformanceSpike(emp sqlc.Employee) RecommendationInput {
	empID := uuidToString(emp.ID)
	actions, err := json.Marshal([]map[string]any{
		{"type": "public_recognition", "params": map[string]any{"employee_id": empID, "message": fmt.Sprintf("Great work by %s — consistent daily check-ins and strong engagement!", emp.Name)}, "label": "Public recognition"},
	})
	if err != nil {
		slog.Error("templatePerformanceSpike: marshal actions failed", "error", err)
	}
	evidence, err := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": emp.Name, "issue": "exceptional_performance"}},
	})
	if err != nil {
		slog.Error("templatePerformanceSpike: marshal evidence failed", "error", err)
	}
	entityType := "employee"
	return RecommendationInput{
		Category:         "people",
		Priority:         "medium",
		Title:            fmt.Sprintf("Recognize %s for exceptional performance", emp.Name),
		Description:      fmt.Sprintf("%s has shown exceptional consistency — submitted check-ins 6+ of the last 7 days. Consider public recognition.", emp.Name),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &empID,
	}
}

func (r *Recommender) templateTaskOverdue(taskTitle string, taskID string, days int, assigneeName string) RecommendationInput {
	priority := "high"
	if days >= 5 {
		priority = "critical"
	}
	actions, err := json.Marshal([]map[string]any{
		{"type": "send_message", "params": map[string]any{"employee_id": "", "message": fmt.Sprintf("Hi, the task '%s' is %d days overdue. Can you provide an update?", taskTitle, days)}, "label": "Notify assignee"},
		{"type": "reassign_task", "params": map[string]any{"task_id": taskID, "reason": fmt.Sprintf("Overdue by %d days", days)}, "label": "Reassign task"},
	})
	if err != nil {
		slog.Error("templateTaskOverdue: marshal actions failed", "error", err)
	}
	evidence, err := json.Marshal(map[string]any{
		"tasks":     []map[string]any{{"id": taskID, "issue": fmt.Sprintf("overdue_%dd", days)}},
		"employees": []map[string]any{{"name": assigneeName, "issue": "task_overdue"}},
	})
	if err != nil {
		slog.Error("templateTaskOverdue: marshal evidence failed", "error", err)
	}
	entityType := "project"
	return RecommendationInput{
		Category:         "project",
		Priority:         priority,
		Title:            fmt.Sprintf("Critical task overdue: %s (%d days)", taskTitle, days),
		Description:      fmt.Sprintf("'%s' assigned to %s is %d days overdue. Consider reassigning or escalating.", taskTitle, assigneeName, days),
		SuggestedActions: actions,
		Evidence:         evidence,
		TargetEntityType: &entityType,
		TargetEntityID:   &taskID,
	}
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}

// recParseUUID parses a UUID string into pgtype.UUID.
// Named differently from api.parseUUID to avoid import conflicts.
func recParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID: %w", err)
	}
	return u, nil
}

func priorityRank(p string) int {
	switch p {
	case "critical":
		return 1
	case "high":
		return 2
	case "medium":
		return 3
	case "low":
		return 4
	default:
		return 5
	}
}

// ---------------------------------------------------------------------------
// storeIfNew — dedup check (entity-level OR org-level), priority comparison, create if new
// ---------------------------------------------------------------------------

func (r *Recommender) storeIfNew(ctx context.Context, tenantID pgtype.UUID, input RecommendationInput, source string) error {
	var entityType pgtype.Text
	if input.TargetEntityType != nil {
		entityType = pgtype.Text{String: *input.TargetEntityType, Valid: true}
	}
	var entityID pgtype.UUID
	if input.TargetEntityID != nil {
		parsed, err := recParseUUID(*input.TargetEntityID)
		if err == nil {
			entityID = parsed
		}
	}

	// Org-level recommendations (NULL entity) use title-based dedup
	if !entityID.Valid {
		dup, err := r.queries.FindDuplicateOrgRecommendation(ctx, sqlc.FindDuplicateOrgRecommendationParams{
			TenantID: tenantID,
			Category: input.Category,
			Title:    input.Title,
		})
		if err == nil && dup.ID.Valid {
			if priorityRank(input.Priority) < priorityRank(dup.Priority) {
				_ = r.queries.UpdateRecommendationStatus(ctx, sqlc.UpdateRecommendationStatusParams{
					ID: dup.ID, TenantID: tenantID, Status: "expired",
				})
			} else {
				return nil
			}
		}
	} else {
		// Entity-level dedup: category + entity_type + entity_id
		dup, err := r.queries.FindDuplicateRecommendation(ctx, sqlc.FindDuplicateRecommendationParams{
			TenantID:         tenantID,
			Category:         input.Category,
			TargetEntityType: entityType,
			TargetEntityID:   entityID,
		})
		if err == nil && dup.ID.Valid {
			if priorityRank(input.Priority) < priorityRank(dup.Priority) {
				_ = r.queries.UpdateRecommendationStatus(ctx, sqlc.UpdateRecommendationStatusParams{
					ID: dup.ID, TenantID: tenantID, Status: "expired",
				})
			} else {
				return nil
			}
		}
	}

	_, err := r.queries.CreateRecommendation(ctx, sqlc.CreateRecommendationParams{
		TenantID:         tenantID,
		Category:         input.Category,
		Priority:         input.Priority,
		Title:            input.Title,
		Description:      input.Description,
		SuggestedActions: input.SuggestedActions,
		Evidence:         input.Evidence,
		Source:           source,
		TargetEntityType: entityType,
		TargetEntityID:   entityID,
		ExpiresAt:        pgtype.Timestamptz{Time: time.Now().Add(72 * time.Hour), Valid: true},
	})
	return err
}

// ---------------------------------------------------------------------------
// DailyScan — expire old recs, gather context, call LLM, parse JSON, store
// ---------------------------------------------------------------------------

// DailyScan runs the daily AI recommendation scan for a tenant.
// It expires old recommendations, gathers company context, asks the LLM for
// up to 5 actionable recommendations, then stores them with dedup.
func (r *Recommender) DailyScan(ctx context.Context, tenantID pgtype.UUID, mentorID, cultureCode string) error {
	slog.Info("recommendation_scan: starting", "tenant", uuidToString(tenantID))

	// Expire old recommendations first
	_ = r.queries.ExpireOldRecommendations(ctx, tenantID)

	// Gather context (reuse ContextService pattern from ExecutionPlanner)
	contextData, err := r.contextService.FormatContextForPrompt(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("gather context: %w", err)
	}

	// Get pending recommendations for dedup hint
	pending, _ := r.queries.GetRecommendationSummary(ctx, tenantID)
	var pendingTitles []string
	for _, p := range pending {
		pendingTitles = append(pendingTitles, p.Title)
	}

	systemPrompt := fmt.Sprintf(`You are an AI management advisor using the %s philosophy.
Based on the following team data, generate up to 5 actionable management recommendations.

Rules:
- Each recommendation must have clear data evidence
- Each must include suggested_actions with valid action types
- Valid action types: schedule_meeting, send_message, create_task, reassign_task, flag_risk, adjust_target, public_recognition, create_suggestion
- Priority: critical (act now) > high (today) > medium (this week) > low (reference)
- Do NOT repeat these pending recommendations: %s
- Use %s cultural communication style

Output a JSON array of recommendations. Each element:
{
  "category": "people|project|kpi|organization",
  "priority": "critical|high|medium|low",
  "title": "short title",
  "description": "2-3 sentences explaining why and impact",
  "suggested_actions": [{"type": "action_type", "params": {...}, "label": "button text"}],
  "evidence": {"signals": [{"name": "...", "value": 0.0}], "employees": [{"name": "...", "issue": "..."}], "metrics": [{"name": "...", "trend": "..."}], "tasks": [{"id": "...", "issue": "..."}]},
  "target_entity_type": "employee|project|metric|goal|null",
  "target_entity_id": "uuid or null"
}`, mentorID, strings.Join(pendingTitles, "; "), cultureCode)

	userPrompt := fmt.Sprintf("## Team Data\n\n%s", contextData)

	// Use ChatLong for 4096 output tokens — daily scan generates up to 5 full recommendations
	response, err := r.llm.ChatLong(ctx, systemPrompt, userPrompt)
	if err != nil {
		slog.Error("recommendation_scan: LLM failed", "error", err)
		return err
	}

	// Parse response — strip markdown code fences if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var recs []RecommendationInput
	if err := json.Unmarshal([]byte(response), &recs); err != nil {
		slog.Warn("recommendation_scan: parse failed", "error", err, "response", response[:min(len(response), 200)])
		return fmt.Errorf("parse LLM response: %w", err)
	}

	stored := 0
	for _, rec := range recs {
		if err := r.storeIfNew(ctx, tenantID, rec, "daily_scan"); err != nil {
			slog.Error("recommendation_scan: store failed", "title", rec.Title, "error", err)
			continue
		}
		stored++
	}

	slog.Info("recommendation_scan: done", "generated", len(recs), "stored", stored)
	return nil
}

// ---------------------------------------------------------------------------
// RealtimeEvaluate — switch on eventType, dispatch to template generators
// ---------------------------------------------------------------------------

// RealtimeEvaluate checks trigger conditions and generates recommendations.
// Called from trigger system, signal generator, and metric handlers.
// Accepts empName+empID (not sqlc.Employee or report.EmployeeInfo) to bridge
// type differences between callers.
func (r *Recommender) RealtimeEvaluate(ctx context.Context, tenantID pgtype.UUID, eventType string, empName string, empID pgtype.UUID, data map[string]any) error {
	// Build a minimal sqlc.Employee for template usage
	emp := sqlc.Employee{ID: empID, Name: empName}

	var input *RecommendationInput

	switch eventType {
	case "consecutive_miss":
		days, _ := data["days"].(int64)
		if days >= 3 {
			rec := r.templateConsecutiveMiss(emp, days)
			input = &rec
		}
	case "sentiment_drop":
		trend, _ := data["trend"].(string)
		rec := r.templateSentimentDrop(emp, trend)
		input = &rec
	case "exceptional_performance":
		rec := r.templatePerformanceSpike(emp)
		input = &rec
	// Complex LLM-based triggers deferred to v2:
	// case "signal_high_score", "blocker_cascade", "metric_anomaly":
	//   These require separate prompt design and cost analysis.
	//   Template triggers cover 80% of real-time use cases.
	default:
		slog.Debug("recommendation: unknown event type", "type", eventType)
	}

	if input == nil {
		return nil
	}

	return r.storeIfNew(ctx, tenantID, *input, "realtime_trigger")
}
