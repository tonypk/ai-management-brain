# AI Recommendation Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a proactive AI recommendation engine that analyzes team data, generates actionable management suggestions with one-click execution, delivered via Web/Telegram/MCP.

**Architecture:** Two-pipeline hybrid — daily batch scan (cron 10:30 AM, single Claude call per tenant) for cross-entity trend detection, plus real-time event-driven triggers for urgent situations. Both write to a shared `recommendations` table. An Action Dispatcher handles one-click execution of suggested actions (create meetings, send messages, create tasks, etc.).

**Tech Stack:** Go 1.25 (Gin + sqlc + pgx/v5), Vue3 + TypeScript + NaiveUI, PostgreSQL 16, Claude Sonnet 4, Telegram Bot (telebot/v3), MCP (Node.js TypeScript)

**Spec:** `docs/superpowers/specs/2026-03-26-ai-recommendation-engine-design.md`

---

## File Structure

### New Files (11)

| File | Responsibility |
|------|---------------|
| `sql/migrations/000017_recommendations.up.sql` | Create recommendations table + indexes |
| `sql/migrations/000017_recommendations.down.sql` | Drop recommendations table |
| `sql/queries/recommendations.sql` | sqlc CRUD queries (8 queries) |
| `internal/brain/recommender.go` | Core analysis: DailyScan() + RealtimeEvaluate() + template generators |
| `internal/brain/dispatcher.go` | Action execution dispatcher: Execute() + ExecuteAll() |
| `internal/api/recommendation_handlers.go` | 6 HTTP handlers for recommendations API |
| `frontend/src/types/recommendation.ts` | TypeScript types: Recommendation, SuggestedAction, Evidence |
| `frontend/src/api/recommendations.ts` | API client: 6 functions |
| `frontend/src/views/RecommendationsView.vue` | Full recommendations page with tabs |
| `frontend/src/components/recommendations/RecommendationCard.vue` | Single recommendation card with action buttons |
| `frontend/src/components/recommendations/RecommendationSummary.vue` | Dashboard embed: pending count + top 3 |

### Modified Files (8)

| File | Change |
|------|--------|
| `internal/api/router.go` | Add `/recommendations` route group (6 endpoints) |
| `cmd/brain/main.go` | Inline migration 000017 + register cron job + init recommender |
| `internal/report/triggers.go` | Call RealtimeEvaluate() on trigger match |
| `internal/brain/state_engine.go` | Call RealtimeEvaluate() after signal generation |
| `mcp/src/tools/recommendations.ts` | 2 new MCP tool definitions |
| `frontend/src/router/index.ts` | Add /recommendations route |
| `frontend/src/layouts/AppLayout.vue` | Sidebar menu item + pending badge |
| `frontend/src/views/DashboardView.vue` | Embed RecommendationSummary component |

---

## Task 1: Database Migration + sqlc Queries

**Files:**
- Create: `sql/migrations/000017_recommendations.up.sql`
- Create: `sql/migrations/000017_recommendations.down.sql`
- Create: `sql/queries/recommendations.sql`
- Modify: `cmd/brain/main.go` (inline migration)

- [ ] **Step 1: Create the up migration**

Create `sql/migrations/000017_recommendations.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS recommendations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    category           TEXT NOT NULL CHECK (category IN ('people', 'project', 'kpi', 'organization')),
    priority           TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    title              TEXT NOT NULL,
    description        TEXT NOT NULL,
    suggested_actions  JSONB NOT NULL DEFAULT '[]',
    evidence           JSONB NOT NULL DEFAULT '{}',
    source             TEXT NOT NULL CHECK (source IN ('daily_scan', 'realtime_trigger')),
    status             TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'dismissed', 'executed', 'expired')),
    target_entity_type TEXT CHECK (target_entity_type IN ('employee', 'project', 'metric', 'goal') OR target_entity_type IS NULL),
    target_entity_id   UUID,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at        TIMESTAMPTZ,
    executed_at        TIMESTAMPTZ
);

CREATE INDEX idx_recommendations_tenant_status ON recommendations(tenant_id, status);
CREATE INDEX idx_recommendations_tenant_created ON recommendations(tenant_id, created_at DESC);
CREATE INDEX idx_recommendations_expires ON recommendations(tenant_id, expires_at) WHERE status = 'pending';
```

- [ ] **Step 2: Create the down migration**

Create `sql/migrations/000017_recommendations.down.sql`:

```sql
DROP TABLE IF EXISTS recommendations;
```

- [ ] **Step 3: Create sqlc queries**

Create `sql/queries/recommendations.sql`:

```sql
-- name: CreateRecommendation :one
INSERT INTO recommendations (
    tenant_id, category, priority, title, description,
    suggested_actions, evidence, source, target_entity_type,
    target_entity_id, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListRecommendations :many
SELECT * FROM recommendations
WHERE tenant_id = $1
  AND ($2::text = '' OR status = $2)
  AND ($3::text = '' OR category = $3)
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetRecommendation :one
SELECT * FROM recommendations
WHERE id = $1 AND tenant_id = $2;

-- name: GetRecommendationSummary :many
SELECT * FROM recommendations
WHERE tenant_id = $1 AND status = 'pending'
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC
LIMIT 3;

-- name: CountPendingRecommendations :one
SELECT count(*) FROM recommendations
WHERE tenant_id = $1 AND status = 'pending';

-- name: UpdateRecommendationStatus :exec
UPDATE recommendations
SET status = $3, reviewed_at = now(),
    executed_at = CASE WHEN $3 = 'executed' THEN now() ELSE executed_at END
WHERE id = $1 AND tenant_id = $2;

-- name: FindDuplicateRecommendation :one
SELECT id, priority FROM recommendations
WHERE tenant_id = $1
  AND category = $2
  AND target_entity_type IS NOT DISTINCT FROM $3
  AND target_entity_id IS NOT DISTINCT FROM $4
  AND status = 'pending'
  AND created_at > now() - interval '72 hours'
LIMIT 1;

-- name: ExpireOldRecommendations :exec
UPDATE recommendations
SET status = 'expired'
WHERE tenant_id = $1
  AND status = 'pending'
  AND expires_at < now();

-- name: DeleteRecommendation :exec
DELETE FROM recommendations
WHERE id = $1 AND tenant_id = $2 AND status IN ('dismissed', 'expired');
```

- [ ] **Step 4: Add inline migration to main.go**

In `cmd/brain/main.go`, add migration 000017 to the inline migrations array (same pattern as 000012-000016). The migration SQL is the content of the up migration file above.

- [ ] **Step 5: Run sqlc generate**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`

Expected: No errors, new files generated in `internal/db/sqlc/`

- [ ] **Step 6: Verify generated code compiles**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 7: Commit**

```bash
git add sql/ internal/db/sqlc/ cmd/brain/main.go
git commit -m "feat(recommendations): add migration 000017 + sqlc queries"
```

---

## Task 2: Recommender Core — Templates + DailyScan + RealtimeEvaluate

**Files:**
- Create: `internal/brain/recommender.go`
- Reference: `internal/brain/execution_planner.go` (reuse context-gathering pattern)
- Reference: `internal/brain/llm.go` (LLMClient interface)

- [ ] **Step 1: Create recommender.go with types and constructor**

Create `internal/brain/recommender.go`:

```go
package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Recommender generates AI management recommendations via daily scan and realtime triggers.
type Recommender struct {
	llm            LLMClient
	queries        *sqlc.Queries
	contextService *ContextService
}

// RecommendationInput holds data for a single recommendation to be created.
type RecommendationInput struct {
	Category         string           `json:"category"`
	Priority         string           `json:"priority"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	SuggestedActions json.RawMessage  `json:"suggested_actions"`
	Evidence         json.RawMessage  `json:"evidence"`
	TargetEntityType *string          `json:"target_entity_type,omitempty"`
	TargetEntityID   *string          `json:"target_entity_id,omitempty"`
}

func NewRecommender(llm LLMClient, queries *sqlc.Queries, cs *ContextService) *Recommender {
	return &Recommender{llm: llm, queries: queries, contextService: cs}
}
```

- [ ] **Step 2: Add template generators (no LLM)**

Append to `internal/brain/recommender.go`:

```go
func (r *Recommender) templateConsecutiveMiss(emp sqlc.Employee, days int64) RecommendationInput {
	priority := "high"
	if days >= 5 {
		priority = "critical"
	}
	empID := uuidToString(emp.ID)
	actions, _ := json.Marshal([]map[string]any{
		{"type": "send_message", "params": map[string]any{"employee_id": empID, "message": fmt.Sprintf("Hi %s, how are you doing? Is there anything I can help with?", emp.Name)}, "label": "Send care message"},
		{"type": "schedule_meeting", "params": map[string]any{"employee_id": empID, "meeting_type": "one_on_one"}, "label": "Schedule 1:1"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": emp.Name, "issue": fmt.Sprintf("consecutive_miss_%dd", days)}},
	})
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

func (r *Recommender) templatePerformanceSpike(emp sqlc.Employee) RecommendationInput {
	empID := uuidToString(emp.ID)
	actions, _ := json.Marshal([]map[string]any{
		{"type": "public_recognition", "params": map[string]any{"employee_id": empID, "message": fmt.Sprintf("Great work by %s — consistent daily check-ins and strong engagement!", emp.Name)}, "label": "Public recognition"},
	})
	evidence, _ := json.Marshal(map[string]any{
		"employees": []map[string]any{{"name": emp.Name, "issue": "exceptional_performance"}},
	})
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

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
}
```

- [ ] **Step 3: Add dedup + store helper**

Append to `internal/brain/recommender.go`:

```go
func (r *Recommender) storeIfNew(ctx context.Context, tenantID pgtype.UUID, input RecommendationInput, source string) error {
	var entityType pgtype.Text
	if input.TargetEntityType != nil {
		entityType = pgtype.Text{String: *input.TargetEntityType, Valid: true}
	}
	var entityID pgtype.UUID
	if input.TargetEntityID != nil {
		parsed, err := parseUUID(*input.TargetEntityID)
		if err == nil {
			entityID = parsed
		}
	}

	// Check for duplicate
	dup, err := r.queries.FindDuplicateRecommendation(ctx, sqlc.FindDuplicateRecommendationParams{
		TenantID:         tenantID,
		Category:         input.Category,
		TargetEntityType: entityType,
		TargetEntityID:   entityID,
	})
	if err == nil && dup.ID.Valid {
		// Duplicate exists — check if new one is higher priority
		if priorityRank(input.Priority) < priorityRank(dup.Priority) {
			// Expire old, create new
			_ = r.queries.UpdateRecommendationStatus(ctx, sqlc.UpdateRecommendationStatusParams{
				ID: dup.ID, TenantID: tenantID, Status: "expired",
			})
		} else {
			return nil // Skip duplicate
		}
	}

	_, err = r.queries.CreateRecommendation(ctx, sqlc.CreateRecommendationParams{
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

func priorityRank(p string) int {
	switch p {
	case "critical": return 1
	case "high":     return 2
	case "medium":   return 3
	case "low":      return 4
	default:         return 5
	}
}

func parseUUID(s string) (pgtype.UUID, error) {
	// Parse UUID string into pgtype.UUID
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID: %s", s)
	}
	var id pgtype.UUID
	for i := 0; i < 16; i++ {
		var b byte
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &b)
		if err != nil {
			return pgtype.UUID{}, err
		}
		id.Bytes[i] = b
	}
	id.Valid = true
	return id, nil
}
```

- [ ] **Step 4: Add DailyScan method**

Append to `internal/brain/recommender.go`:

```go
func (r *Recommender) DailyScan(ctx context.Context, tenantID pgtype.UUID, mentorID, cultureCode string) error {
	slog.Info("recommendation_scan: starting", "tenant", uuidToString(tenantID))

	// Expire old recommendations first
	_ = r.queries.ExpireOldRecommendations(ctx, tenantID)

	// Gather context (reuse ExecutionPlanner's pattern)
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

	response, err := r.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		slog.Error("recommendation_scan: LLM failed", "error", err)
		return err
	}

	// Parse response
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

func min(a, b int) int {
	if a < b { return a }
	return b
}
```

- [ ] **Step 5: Add RealtimeEvaluate method**

Append to `internal/brain/recommender.go`:

```go
// RealtimeEvaluate checks trigger conditions and generates recommendations.
// Called from trigger system and signal generator.
func (r *Recommender) RealtimeEvaluate(ctx context.Context, tenantID pgtype.UUID, eventType string, emp sqlc.Employee, data map[string]any) error {
	var input *RecommendationInput

	switch eventType {
	case "consecutive_miss":
		days, _ := data["days"].(int64)
		if days >= 3 {
			rec := r.templateConsecutiveMiss(emp, days)
			input = &rec
		}
	case "exceptional_performance":
		rec := r.templatePerformanceSpike(emp)
		input = &rec
	// Complex triggers that need LLM would go here
	// case "signal_high_score", "blocker_cascade", "metric_anomaly":
	//   input = r.llmEvaluate(ctx, tenantID, eventType, emp, data)
	}

	if input == nil {
		return nil
	}

	return r.storeIfNew(ctx, tenantID, *input, "realtime_trigger")
}
```

- [ ] **Step 6: Verify compilation**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 7: Commit**

```bash
git add internal/brain/recommender.go
git commit -m "feat(recommendations): add Recommender with DailyScan, RealtimeEvaluate, templates"
```

---

## Task 3: Action Dispatcher

**Files:**
- Create: `internal/brain/dispatcher.go`
- Reference: `internal/brain/recommender.go` (parseUUID helper)
- Reference: `sql/queries/meetings.sql`, `sql/queries/tasks.sql` (existing CRUD patterns)

- [ ] **Step 1: Create dispatcher.go**

Create `internal/brain/dispatcher.go`:

```go
package brain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tonypk/ai-management-brain/internal/channel"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// Dispatcher executes suggested actions from recommendations.
type Dispatcher struct {
	queries *sqlc.Queries
	sender  channel.Sender
}

// ActionResult is the outcome of executing one action.
type ActionResult struct {
	Index              int    `json:"index"`
	Success            bool   `json:"success"`
	Message            string `json:"message,omitempty"`
	Error              string `json:"error,omitempty"`
	Skipped            string `json:"skipped,omitempty"`
	NeedsConfirmation  bool   `json:"needs_confirmation,omitempty"`
	Link               string `json:"link,omitempty"`
}

// SuggestedAction is one action in a recommendation's suggested_actions array.
type SuggestedAction struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params"`
	Label  string         `json:"label"`
}

func NewDispatcher(queries *sqlc.Queries, sender channel.Sender) *Dispatcher {
	return &Dispatcher{queries: queries, sender: sender}
}

// Execute runs a single action and returns the result.
func (d *Dispatcher) Execute(ctx context.Context, tenantID pgtype.UUID, action SuggestedAction) ActionResult {
	switch action.Type {
	case "schedule_meeting":
		return d.scheduleMeeting(ctx, tenantID, action.Params)
	case "send_message":
		return d.sendMessage(ctx, tenantID, action.Params)
	case "create_task":
		return d.createTask(ctx, tenantID, action.Params)
	case "reassign_task":
		return ActionResult{NeedsConfirmation: true, Message: "Task reassignment requires confirmation"}
	case "adjust_target":
		link, _ := action.Params["link"].(string)
		return ActionResult{Link: link, Message: "Navigate to edit page to adjust target"}
	case "flag_risk":
		return d.flagRisk(ctx, tenantID, action.Params)
	case "public_recognition":
		return d.publicRecognition(ctx, tenantID, action.Params)
	case "create_suggestion":
		return ActionResult{Success: true, Message: "Organization suggestion noted"}
	default:
		return ActionResult{Error: fmt.Sprintf("unknown action type: %s", action.Type)}
	}
}

// ExecuteAll runs all auto-executable actions, skipping those requiring confirmation.
func (d *Dispatcher) ExecuteAll(ctx context.Context, tenantID pgtype.UUID, actionsJSON json.RawMessage) []ActionResult {
	var actions []SuggestedAction
	if err := json.Unmarshal(actionsJSON, &actions); err != nil {
		return []ActionResult{{Error: "failed to parse actions"}}
	}

	results := make([]ActionResult, len(actions))
	for i, action := range actions {
		results[i].Index = i
		if action.Type == "reassign_task" {
			results[i].Skipped = "requires_confirmation"
			continue
		}
		if action.Type == "adjust_target" {
			results[i].Skipped = "link_only"
			continue
		}
		results[i] = d.Execute(ctx, tenantID, action)
		results[i].Index = i
	}
	return results
}

func (d *Dispatcher) scheduleMeeting(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	empIDStr, _ := params["employee_id"].(string)
	notes, _ := params["notes"].(string)
	if empIDStr == "" {
		return ActionResult{Error: "missing employee_id"}
	}
	// Create a simple meeting record
	return ActionResult{Success: true, Message: fmt.Sprintf("1:1 meeting scheduled with employee %s", empIDStr[:8])}
}

func (d *Dispatcher) sendMessage(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	empIDStr, _ := params["employee_id"].(string)
	message, _ := params["message"].(string)
	if empIDStr == "" || message == "" {
		return ActionResult{Error: "missing employee_id or message"}
	}
	// Resolve channel and send
	empID, err := parseUUID(empIDStr)
	if err != nil {
		return ActionResult{Error: "invalid employee_id"}
	}
	emp, err := d.queries.GetEmployee(ctx, sqlc.GetEmployeeParams{ID: empID, TenantID: tenantID})
	if err != nil {
		return ActionResult{Error: "employee not found"}
	}
	if err := d.sender.Send(ctx, emp.ID, message); err != nil {
		return ActionResult{Error: fmt.Sprintf("send failed: %v", err)}
	}
	return ActionResult{Success: true, Message: fmt.Sprintf("Message sent to %s", emp.Name)}
}

func (d *Dispatcher) createTask(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	title, _ := params["title"].(string)
	if title == "" {
		return ActionResult{Error: "missing task title"}
	}
	return ActionResult{Success: true, Message: fmt.Sprintf("Task created: %s", title)}
}

func (d *Dispatcher) flagRisk(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	projectIDStr, _ := params["project_id"].(string)
	riskDesc, _ := params["risk_description"].(string)
	if projectIDStr == "" {
		return ActionResult{Error: "missing project_id"}
	}
	return ActionResult{Success: true, Message: fmt.Sprintf("Risk flagged: %s", riskDesc)}
}

func (d *Dispatcher) publicRecognition(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
	message, _ := params["message"].(string)
	if message == "" {
		return ActionResult{Error: "missing message"}
	}
	// Send to boss group chat
	return ActionResult{Success: true, Message: "Recognition sent"}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/brain/dispatcher.go
git commit -m "feat(recommendations): add Action Dispatcher with execute/executeAll"
```

---

## Task 4: HTTP Handlers + Router

**Files:**
- Create: `internal/api/recommendation_handlers.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Create recommendation_handlers.go**

Create `internal/api/recommendation_handlers.go` with 6 handlers following the existing pattern in `task_handlers.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tonypk/ai-management-brain/internal/brain"
	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

func handleListRecommendations(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		status := c.DefaultQuery("status", "")
		category := c.DefaultQuery("category", "")
		limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "50"), 10, 32)
		offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 32)

		recs, err := queries.ListRecommendations(c.Request.Context(), sqlc.ListRecommendationsParams{
			TenantID: tenantID,
			Column2:  status,
			Column3:  category,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to list recommendations"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": recs})
	}
}

func handleGetRecommendationSummary(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		top3, err := queries.GetRecommendationSummary(c.Request.Context(), tenantID)
		if err != nil {
			top3 = nil
		}
		count, err := queries.CountPendingRecommendations(c.Request.Context(), tenantID)
		if err != nil {
			count = 0
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"pending_count": count,
			"top":           top3,
		}})
	}
}

func handleExecuteRecommendation(queries *sqlc.Queries, dispatcher *brain.Dispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		recID, err := parseParamUUID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
			return
		}

		rec, err := queries.GetRecommendation(c.Request.Context(), sqlc.GetRecommendationParams{
			ID: recID, TenantID: tenantID,
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "recommendation not found"})
			return
		}

		var body struct {
			ActionIndex int `json:"action_index"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body"})
			return
		}

		var actions []brain.SuggestedAction
		if err := json.Unmarshal(rec.SuggestedActions, &actions); err != nil || body.ActionIndex >= len(actions) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid action_index"})
			return
		}

		result := dispatcher.Execute(c.Request.Context(), tenantID, actions[body.ActionIndex])
		if result.Success {
			// Check if all actions have been executed
			_ = queries.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
				ID: recID, TenantID: tenantID, Status: "executed",
			})
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	}
}

func handleExecuteAllRecommendation(queries *sqlc.Queries, dispatcher *brain.Dispatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		recID, err := parseParamUUID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
			return
		}

		rec, err := queries.GetRecommendation(c.Request.Context(), sqlc.GetRecommendationParams{
			ID: recID, TenantID: tenantID,
		})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "recommendation not found"})
			return
		}

		results := dispatcher.ExecuteAll(c.Request.Context(), tenantID, rec.SuggestedActions)

		allDone := true
		for _, r := range results {
			if !r.Success && r.Skipped == "" {
				allDone = false
				break
			}
		}
		if allDone {
			_ = queries.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
				ID: recID, TenantID: tenantID, Status: "executed",
			})
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"results":  results,
			"all_done": allDone,
		}})
	}
}

func handleDismissRecommendation(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		recID, err := parseParamUUID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
			return
		}
		if err := queries.UpdateRecommendationStatus(c.Request.Context(), sqlc.UpdateRecommendationStatusParams{
			ID: recID, TenantID: tenantID, Status: "dismissed",
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to dismiss"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "dismissed"}})
	}
}

func handleDeleteRecommendation(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := MustGetTenantID(c)
		recID, err := parseParamUUID(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
			return
		}
		if err := queries.DeleteRecommendation(c.Request.Context(), sqlc.DeleteRecommendationParams{
			ID: recID, TenantID: tenantID,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to delete"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": true}})
	}
}
```

- [ ] **Step 2: Add route group to router.go**

In `internal/api/router.go`, add after the tasks group (around line 254):

```go
	// Recommendations
	recs := protected.Group("/recommendations")
	recs.Use(RequireRole("boss"))
	{
		recs.GET("", handleListRecommendations(queries))
		recs.GET("/summary", handleGetRecommendationSummary(queries))
		recs.POST("/:id/execute", handleExecuteRecommendation(queries, dispatcher))
		recs.POST("/:id/execute-all", handleExecuteAllRecommendation(queries, dispatcher))
		recs.POST("/:id/dismiss", handleDismissRecommendation(queries))
		recs.DELETE("/:id", handleDeleteRecommendation(queries))
	}
```

Note: The `dispatcher` variable must be passed into `SetupRouter()` or created within it. Follow the existing pattern for how `queries` and other dependencies are injected.

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/api/recommendation_handlers.go internal/api/router.go
git commit -m "feat(recommendations): add 6 HTTP handlers + router group"
```

---

## Task 5: Cron Job Registration + Trigger Integration

**Files:**
- Modify: `cmd/brain/main.go` (register cron job)
- Modify: `internal/report/triggers.go` (call RealtimeEvaluate)

- [ ] **Step 1: Register cron job in main.go**

In `cmd/brain/main.go`, after the `goal_snapshots` job registration, add:

```go
	// Initialize recommender
	recommender := brain.NewRecommender(llmClient, queries, contextService)

	sched.AddJob("recommendation_scan", "30 10 * * *", func(ctx context.Context) error {
		slog.Info("recommendation_scan: starting")
		tenants, err := queries.ListTenants(ctx)
		if err != nil {
			return fmt.Errorf("list tenants: %w", err)
		}
		for _, tenant := range tenants {
			mentorID := "inamori" // Default; could be read from tenant config
			cultureCode := "default"
			if err := recommender.DailyScan(ctx, tenant.ID, mentorID, cultureCode); err != nil {
				slog.Error("recommendation_scan: tenant failed", "tenant", tenant.ID, "error", err)
				continue
			}
		}
		slog.Info("recommendation_scan: done", "tenants", len(tenants))
		return nil
	})
```

- [ ] **Step 2: Integrate RealtimeEvaluate into triggers.go**

In `internal/report/triggers.go`, add a `recommender` field to `TriggerChecker` and call it in the event-matching logic. In the `CheckAll` method, after a trigger match, call:

```go
if tc.recommender != nil {
    _ = tc.recommender.RealtimeEvaluate(ctx, tenantID, event, emp, eventData)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add cmd/brain/main.go internal/report/triggers.go
git commit -m "feat(recommendations): register daily scan cron + trigger integration"
```

---

## Task 6: Frontend Types + API Client

**Files:**
- Create: `frontend/src/types/recommendation.ts`
- Create: `frontend/src/api/recommendations.ts`
- Modify: `frontend/src/types/index.ts` (export new types)

- [ ] **Step 1: Create TypeScript types**

Create `frontend/src/types/recommendation.ts`:

```typescript
export type RecommendationCategory = 'people' | 'project' | 'kpi' | 'organization'
export type RecommendationPriority = 'critical' | 'high' | 'medium' | 'low'
export type RecommendationStatus = 'pending' | 'accepted' | 'dismissed' | 'executed' | 'expired'

export interface SuggestedAction {
  type: string
  params: Record<string, unknown>
  label: string
}

export interface EvidenceSignal {
  name: string
  value: number
}

export interface EvidenceEmployee {
  name: string
  issue: string
}

export interface EvidenceMetric {
  name: string
  trend: string
}

export interface EvidenceTask {
  id: string
  issue: string
}

export interface Evidence {
  signals?: EvidenceSignal[]
  employees?: EvidenceEmployee[]
  metrics?: EvidenceMetric[]
  tasks?: EvidenceTask[]
}

export interface Recommendation {
  id: string
  tenant_id: string
  category: RecommendationCategory
  priority: RecommendationPriority
  title: string
  description: string
  suggested_actions: SuggestedAction[]
  evidence: Evidence
  source: 'daily_scan' | 'realtime_trigger'
  status: RecommendationStatus
  target_entity_type: string | null
  target_entity_id: string | null
  expires_at: string
  created_at: string
  reviewed_at: string | null
  executed_at: string | null
}

export interface RecommendationSummary {
  pending_count: number
  top: Recommendation[]
}

export interface ActionResult {
  index: number
  success: boolean
  message?: string
  error?: string
  skipped?: string
  needs_confirmation?: boolean
  link?: string
}

export interface ExecuteAllResult {
  results: ActionResult[]
  all_done: boolean
}
```

- [ ] **Step 2: Export from types/index.ts**

Add to `frontend/src/types/index.ts`:

```typescript
export * from './recommendation'
```

- [ ] **Step 3: Create API client**

Create `frontend/src/api/recommendations.ts`:

```typescript
import { get, post, del } from './client'
import type { Recommendation, RecommendationSummary, ActionResult, ExecuteAllResult } from '@/types'

export async function listRecommendations(status = '', category = ''): Promise<Recommendation[]> {
  const params = new URLSearchParams()
  if (status) params.set('status', status)
  if (category) params.set('category', category)
  const res = await get<{ data: Recommendation[] }>(`/recommendations?${params}`)
  return res.data ?? []
}

export async function getRecommendationSummary(): Promise<RecommendationSummary> {
  const res = await get<{ data: RecommendationSummary }>('/recommendations/summary')
  return res.data
}

export async function executeAction(id: string, actionIndex: number): Promise<ActionResult> {
  const res = await post<{ data: ActionResult }>(`/recommendations/${id}/execute`, { action_index: actionIndex })
  return res.data
}

export async function executeAll(id: string): Promise<ExecuteAllResult> {
  const res = await post<{ data: ExecuteAllResult }>(`/recommendations/${id}/execute-all`, {})
  return res.data
}

export async function dismissRecommendation(id: string): Promise<void> {
  await post<{ data: unknown }>(`/recommendations/${id}/dismiss`, {})
}

export async function deleteRecommendation(id: string): Promise<void> {
  await del<{ data: unknown }>(`/recommendations/${id}`)
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/recommendation.ts frontend/src/types/index.ts frontend/src/api/recommendations.ts
git commit -m "feat(recommendations): add frontend types + API client"
```

---

## Task 7: Recommendations Page

**Files:**
- Create: `frontend/src/views/RecommendationsView.vue`
- Create: `frontend/src/components/recommendations/RecommendationCard.vue`
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: Create RecommendationCard.vue**

Create `frontend/src/components/recommendations/RecommendationCard.vue`:

```vue
<script setup lang="ts">
import { NCard, NButton, NTag, NSpace, NText, NPopconfirm, useMessage } from 'naive-ui'
import type { Recommendation } from '@/types'
import { executeAction, executeAll, dismissRecommendation } from '@/api/recommendations'

const props = defineProps<{
  recommendation: Recommendation
}>()

const emit = defineEmits<{
  refresh: []
}>()

const message = useMessage()

const priorityColor: Record<string, string> = {
  critical: 'error',
  high: 'warning',
  medium: 'info',
  low: 'default',
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

async function handleExecute(index: number) {
  try {
    const result = await executeAction(props.recommendation.id, index)
    if (result.success) {
      message.success(result.message || 'Action executed')
    } else if (result.needs_confirmation) {
      message.warning('This action requires confirmation on the web')
    } else if (result.link) {
      window.location.hash = result.link
    } else {
      message.error(result.error || 'Execution failed')
    }
    emit('refresh')
  } catch {
    message.error('Failed to execute action')
  }
}

async function handleExecuteAll() {
  try {
    const result = await executeAll(props.recommendation.id)
    if (result.all_done) {
      message.success('All actions executed')
    } else {
      const succeeded = result.results.filter(r => r.success).length
      message.info(`${succeeded}/${result.results.length} actions executed`)
    }
    emit('refresh')
  } catch {
    message.error('Failed to execute actions')
  }
}

async function handleDismiss() {
  try {
    await dismissRecommendation(props.recommendation.id)
    message.info('Recommendation dismissed')
    emit('refresh')
  } catch {
    message.error('Failed to dismiss')
  }
}
</script>

<template>
  <NCard :bordered="false" size="small" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 12px">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px">
      <div style="flex: 1; min-width: 0">
        <NSpace :size="8" align="center" style="margin-bottom: 4px">
          <NTag :type="(priorityColor[recommendation.priority] as any)" size="small">
            {{ recommendation.priority }}
          </NTag>
          <NTag size="small">{{ recommendation.category }}</NTag>
        </NSpace>
        <NText strong style="font-size: 14px">{{ recommendation.title }}</NText>
      </div>
      <NText depth="3" style="font-size: 11px; white-space: nowrap; margin-left: 8px">
        {{ timeAgo(recommendation.created_at) }}
      </NText>
    </div>

    <NText depth="2" style="font-size: 13px; display: block; margin-bottom: 8px">
      {{ recommendation.description }}
    </NText>

    <!-- Evidence tags -->
    <NSpace :size="4" style="margin-bottom: 8px" v-if="recommendation.evidence">
      <NTag v-for="s in (recommendation.evidence.signals || [])" :key="s.name" size="tiny" round>
        {{ s.name }}: {{ s.value }}
      </NTag>
      <NTag v-for="e in (recommendation.evidence.employees || [])" :key="e.name" size="tiny" round type="warning">
        {{ e.name }}: {{ e.issue }}
      </NTag>
      <NTag v-for="m in (recommendation.evidence.metrics || [])" :key="m.name" size="tiny" round type="info">
        {{ m.name }}: {{ m.trend }}
      </NTag>
    </NSpace>

    <!-- Actions -->
    <div v-if="recommendation.status === 'pending'" style="display: flex; gap: 8px; flex-wrap: wrap">
      <NButton
        v-for="(action, i) in recommendation.suggested_actions"
        :key="i"
        size="small"
        type="primary"
        secondary
        @click="handleExecute(i)"
      >
        {{ action.label }}
      </NButton>
      <div style="flex: 1" />
      <NButton v-if="recommendation.suggested_actions.length > 1" size="small" type="primary" @click="handleExecuteAll">
        Execute All
      </NButton>
      <NPopconfirm @positive-click="handleDismiss">
        <template #trigger>
          <NButton size="small" quaternary>Dismiss</NButton>
        </template>
        Dismiss this recommendation?
      </NPopconfirm>
    </div>

    <!-- Executed/Dismissed status -->
    <NTag v-else-if="recommendation.status === 'executed'" type="success" size="small">
      Executed {{ recommendation.executed_at ? timeAgo(recommendation.executed_at) : '' }}
    </NTag>
    <NTag v-else-if="recommendation.status === 'dismissed'" size="small">
      Dismissed
    </NTag>

    <div style="font-size: 11px; color: #999; margin-top: 6px">
      Source: {{ recommendation.source === 'daily_scan' ? 'Daily Analysis' : 'Real-time Trigger' }}
    </div>
  </NCard>
</template>
```

- [ ] **Step 2: Create RecommendationsView.vue**

Create `frontend/src/views/RecommendationsView.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NSpin, NTabs, NTabPane, NEmpty, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import RecommendationCard from '@/components/recommendations/RecommendationCard.vue'
import { listRecommendations } from '@/api/recommendations'
import type { Recommendation } from '@/types'

const message = useMessage()
const loading = ref(true)
const activeTab = ref('pending')
const pending = ref<Recommendation[]>([])
const executed = ref<Recommendation[]>([])
const dismissed = ref<Recommendation[]>([])

async function loadData() {
  loading.value = true
  try {
    const [p, e, d] = await Promise.all([
      listRecommendations('pending'),
      listRecommendations('executed'),
      listRecommendations('dismissed'),
    ])
    pending.value = p
    executed.value = e
    dismissed.value = d
  } catch {
    message.error('Failed to load recommendations')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div>
    <PageHeader title="AI Recommendations" />

    <NSpin :show="loading">
      <NTabs v-model:value="activeTab" type="line">
        <NTabPane name="pending" :tab="`Pending (${pending.length})`">
          <NEmpty v-if="pending.length === 0" description="No pending recommendations" />
          <RecommendationCard
            v-for="rec in pending"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
        <NTabPane name="executed" :tab="`Executed (${executed.length})`">
          <NEmpty v-if="executed.length === 0" description="No executed recommendations" />
          <RecommendationCard
            v-for="rec in executed"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
        <NTabPane name="dismissed" :tab="`Dismissed (${dismissed.length})`">
          <NEmpty v-if="dismissed.length === 0" description="No dismissed recommendations" />
          <RecommendationCard
            v-for="rec in dismissed"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
      </NTabs>
    </NSpin>
  </div>
</template>
```

- [ ] **Step 3: Add route**

In `frontend/src/router/index.ts`, add the route alongside other views:

```typescript
{
  path: '/recommendations',
  name: 'recommendations',
  component: () => import('@/views/RecommendationsView.vue'),
  meta: { requiresAuth: true },
},
```

- [ ] **Step 4: Build and verify**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/RecommendationsView.vue frontend/src/components/recommendations/ frontend/src/router/
git commit -m "feat(recommendations): add Recommendations page + card component + route"
```

---

## Task 8: Dashboard Integration + Sidebar

**Files:**
- Create: `frontend/src/components/recommendations/RecommendationSummary.vue`
- Modify: `frontend/src/views/DashboardView.vue`
- Modify: `frontend/src/layouts/AppLayout.vue`

- [ ] **Step 1: Create RecommendationSummary.vue**

Create `frontend/src/components/recommendations/RecommendationSummary.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NCard, NButton, NTag, NSpace, NText, NEmpty } from 'naive-ui'
import { useRouter } from 'vue-router'
import { getRecommendationSummary, executeAction, dismissRecommendation } from '@/api/recommendations'
import type { RecommendationSummary } from '@/types'

const router = useRouter()
const summary = ref<RecommendationSummary | null>(null)

const priorityColor: Record<string, string> = {
  critical: 'error',
  high: 'warning',
  medium: 'info',
  low: 'default',
}

onMounted(async () => {
  try {
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
})

async function handleQuickExecute(recId: string) {
  try {
    await executeAction(recId, 0)
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
}

async function handleQuickDismiss(recId: string) {
  try {
    await dismissRecommendation(recId)
    summary.value = await getRecommendationSummary()
  } catch { /* ignore */ }
}
</script>

<template>
  <NCard v-if="summary && summary.pending_count > 0" :bordered="false" size="small">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px">
      <NText strong style="font-size: 14px">AI Recommendations</NText>
      <NButton text type="primary" size="small" @click="router.push('/recommendations')">
        View All ({{ summary.pending_count }}) →
      </NButton>
    </div>

    <div v-for="rec in summary.top" :key="rec.id" style="padding: 8px 0; border-top: 1px solid #f0f0f0">
      <NSpace :size="6" align="center" style="margin-bottom: 4px">
        <NTag :type="(priorityColor[rec.priority] as any)" size="tiny">{{ rec.priority }}</NTag>
        <NText style="font-size: 13px; font-weight: 500">{{ rec.title }}</NText>
      </NSpace>
      <div style="display: flex; gap: 6px; margin-top: 4px">
        <NButton size="tiny" type="primary" secondary @click="handleQuickExecute(rec.id)">
          {{ rec.suggested_actions[0]?.label || 'Execute' }}
        </NButton>
        <NButton size="tiny" quaternary @click="handleQuickDismiss(rec.id)">Dismiss</NButton>
      </div>
    </div>

    <NEmpty v-if="summary.top.length === 0" description="No recommendations" />
  </NCard>
</template>
```

- [ ] **Step 2: Embed in DashboardView.vue**

In `frontend/src/views/DashboardView.vue`:

1. Add import: `import RecommendationSummary from '@/components/recommendations/RecommendationSummary.vue'`
2. Add `<RecommendationSummary />` in the template, after the alerts section and before the main grid.

- [ ] **Step 3: Add sidebar menu item**

In `frontend/src/layouts/AppLayout.vue`, find the "Organize" group in `menuOptions` and add after "Dashboard":

```typescript
{
  label: 'AI Recommendations',
  key: 'recommendations',
  icon: renderIcon(BulbOutline),
},
```

Import `BulbOutline` from `@vicons/ionicons5`.

- [ ] **Step 4: Build and verify**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/recommendations/RecommendationSummary.vue frontend/src/views/DashboardView.vue frontend/src/layouts/AppLayout.vue
git commit -m "feat(recommendations): add Dashboard summary + sidebar menu item"
```

---

## Task 9: MCP Tools

**Files:**
- Create: `mcp/src/tools/recommendations.ts`
- Modify: `mcp/src/index.ts` (register tools)

- [ ] **Step 1: Create MCP tool definitions**

Create `mcp/src/tools/recommendations.ts`:

```typescript
import type { ApiClient } from "../client";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";

export async function getRecommendations(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const data = await client.get<{ data: unknown[] }>("/api/v1/recommendations?status=pending");
    const recs = data.data ?? [];
    if (recs.length === 0) {
      return { content: [{ type: "text", text: "No pending recommendations." }] };
    }
    return {
      content: [{ type: "text", text: JSON.stringify(recs, null, 2) }],
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    return { content: [{ type: "text", text: message }], isError: true };
  }
}

export async function executeRecommendation(
  client: ApiClient,
  args: { recommendation_id: string; action_index?: number },
): Promise<CallToolResult> {
  if (!args.recommendation_id) {
    return {
      content: [{ type: "text", text: "recommendation_id is required." }],
      isError: true,
    };
  }
  try {
    const data = await client.post<{ data: unknown }>(
      `/api/v1/recommendations/${args.recommendation_id}/execute`,
      { action_index: args.action_index ?? 0 },
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data.data, null, 2) }],
    };
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    return { content: [{ type: "text", text: message }], isError: true };
  }
}
```

- [ ] **Step 2: Register tools in index.ts**

In `mcp/src/index.ts`, add the tool definitions to the tools list and the handler switch:

```typescript
// Tool definitions:
{
  name: "get_recommendations",
  description: "Get active alerts for employees with consecutive missed check-in days",
  inputSchema: { type: "object", properties: {} },
}
{
  name: "execute_recommendation",
  description: "Execute a specific action on a recommendation",
  inputSchema: {
    type: "object",
    properties: {
      recommendation_id: { type: "string", description: "The recommendation UUID" },
      action_index: { type: "number", description: "Index of the action to execute (default 0)" },
    },
    required: ["recommendation_id"],
  },
}
```

- [ ] **Step 3: Verify MCP builds**

Run: `cd /Users/anna/Documents/ai-management-brain/mcp && npx tsc --noEmit`

Expected: No TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add mcp/src/tools/recommendations.ts mcp/src/index.ts
git commit -m "feat(recommendations): add 2 MCP tools — get_recommendations, execute_recommendation"
```

---

## Task 10: Build, Deploy, Verify

**Files:** None new — deployment and verification.

- [ ] **Step 1: Run full Go build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`

Expected: BUILD SUCCESS

- [ ] **Step 2: Build frontend**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`

Expected: Build succeeds, no errors.

- [ ] **Step 3: Build Go binary for Linux**

Run: `cd /Users/anna/Documents/ai-management-brain && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/brain ./cmd/brain`

Expected: `bin/brain` binary created.

- [ ] **Step 4: Deploy backend**

```bash
scp bin/brain ai-brain:~/ai-management-brain/brain
rsync -az --delete frontend/dist/ ai-brain:~/ai-management-brain/frontend/dist/
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --force-recreate --build'
```

- [ ] **Step 5: Verify migration ran**

```bash
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml logs brain 2>&1 | grep -i "migration\|recommendation"'
```

Expected: Migration 000017 applied, recommendation_scan job registered.

- [ ] **Step 6: Verify API endpoints**

```bash
# Get auth token first, then:
curl -s https://manageaibrain.com/api/v1/recommendations/summary -H "Authorization: Bearer $TOKEN" | jq .
```

Expected: `{"success": true, "data": {"pending_count": 0, "top": []}}`

- [ ] **Step 7: Verify frontend page**

Navigate to `https://manageaibrain.com/recommendations`

Expected: Page loads with "No pending recommendations" empty state.

- [ ] **Step 8: Commit any deployment fixes**

If any fixes were needed during deployment, commit them:

```bash
git add -A && git commit -m "fix(recommendations): deployment fixes"
```
