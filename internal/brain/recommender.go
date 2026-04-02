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

// WorldModelContextProvider provides formatted World Model data for prompt injection.
// Implemented by worldmodel.Service to avoid import cycles (worldmodel imports brain).
type WorldModelContextProvider interface {
	ForRecommenderPrompt(ctx context.Context, tenantID pgtype.UUID) (string, error)
}

// Recommender generates AI management recommendations via daily scan and realtime triggers.
// Uses *AnthropicClient directly (not LLMClient interface) to access ChatLong() for daily scan.
type Recommender struct {
	llm            *AnthropicClient
	queries        *sqlc.Queries
	contextService *ContextService
	memEval        *MemoryPatternEvaluator
	wmService      WorldModelContextProvider
}

// SetMemoryEvaluator injects the memory pattern evaluator after construction.
func (r *Recommender) SetMemoryEvaluator(eval *MemoryPatternEvaluator) {
	r.memEval = eval
}

// SetWorldModelService injects the World Model service for recommendation context.
func (r *Recommender) SetWorldModelService(svc WorldModelContextProvider) {
	r.wmService = svc
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
Based on the following team data and employee memory insights, generate up to 5 actionable management recommendations.

Rules:
- Each recommendation must have clear data evidence
- When memory_highlights are available, use employee behavioral patterns to generate deeper insights
- Include relevant memory evidence in the "evidence" field as "memory_evidence" array
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
  "evidence": {"signals": [...], "employees": [...], "metrics": [...], "tasks": [...], "memory_evidence": [{"date": "YYYY-MM-DD", "content": "memory text", "importance": 0.8}]},
  "target_entity_type": "employee|project|metric|goal|null",
  "target_entity_id": "uuid or null"
}`, mentorID, strings.Join(pendingTitles, "; "), cultureCode)

	userPrompt := fmt.Sprintf("## Team Data\n\n%s", contextData)

	// Inject World Model context as 9th data source
	if r.wmService != nil {
		wmText, wmErr := r.wmService.ForRecommenderPrompt(ctx, tenantID)
		if wmErr == nil && wmText != "" {
			userPrompt += "\n\n## Team World Model Analysis\n\n" + wmText
			userPrompt += "\nBased on the World Model data above, also check for:\n" +
				"1. Knowledge Silos (bus factor=1): recommend knowledge sharing or cross-training\n" +
				"2. Collaboration Gaps: if people work on related problems without collaborating, recommend pairing\n" +
				"3. Skill-Blocker Matches: if someone's blocker matches another's expertise, recommend pairing\n" +
				"4. Escalating Blockers: if a blocker recurs 3+ times, recommend escalation or process change\n" +
				"5. Growth Opportunities: if someone recently leveled up a skill, suggest stretch assignments\n" +
				"6. Risk Patterns: surface team-level risks from AI insights\n\n" +
				"Include \"world_model_evidence\" in the evidence field when generating recommendations from these patterns."
		}
	}

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
	case "memory_extraction_complete":
		// After memory extraction, evaluate patterns and store any recommendations
		if r.memEval != nil {
			recs := r.memEval.EvaluateAfterExtraction(ctx, uuidToString(tenantID), uuidToString(empID), empName)
			for _, rec := range recs {
				if err := r.storeIfNew(ctx, tenantID, rec, "memory_trigger"); err != nil {
					slog.Error("recommendation: memory trigger store failed", "title", rec.Title, "error", err)
				}
			}
		}
		return nil
	default:
		slog.Debug("recommendation: unknown event type", "type", eventType)
	}

	if input == nil {
		return nil
	}

	return r.storeIfNew(ctx, tenantID, *input, "realtime_trigger")
}

// StoreRecommendationIfNew stores a recommendation with dedup check.
// Used by external trigger evaluators (e.g., world model triggers) that can't
// call RealtimeEvaluate directly due to import cycles.
func (r *Recommender) StoreRecommendationIfNew(ctx context.Context, tenantID pgtype.UUID, input RecommendationInput, source string) error {
	return r.storeIfNew(ctx, tenantID, input, source)
}
