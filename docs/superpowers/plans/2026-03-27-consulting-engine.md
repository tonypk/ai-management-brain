# Consulting Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a conversational consulting engine that walks bosses through diagnosis → plan → execute → track, turning AI Management Brain into a McKinsey-style management advisor with execution follow-through.

**Architecture:** Thin engagement layer wrapping existing ContextService, ExecutionPlanner, Dispatcher, and Memory. New `engagements` + `engagement_actions` tables. Fix 3 dispatcher stubs. New bot commands + MCP tools + scheduler job.

**Tech Stack:** Go 1.25 (Gin+sqlc+pgx), PostgreSQL 16, Redis 7, telebot/v3, Claude Sonnet 4

**Spec:** `docs/superpowers/specs/2026-03-27-consulting-engine-design.md`

---

### Task 1: Database Migration + sqlc Queries

**Files:**
- Create: `sql/migrations/000019_engagements.up.sql`
- Create: `sql/migrations/000019_engagements.down.sql`
- Create: `sql/queries/engagements.sql`
- Regenerate: `internal/db/sqlc/` (run sqlc generate)

- [ ] **Step 1: Write migration up**

Create `sql/migrations/000019_engagements.up.sql`:

```sql
-- Migration 000019: Consulting Engagements

CREATE TABLE engagements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  problem_statement TEXT NOT NULL,
  tier TEXT NOT NULL DEFAULT 'standard',
  category TEXT DEFAULT 'general',
  phase TEXT NOT NULL DEFAULT 'intake',
  diagnosis_questions JSONB DEFAULT '[]'::jsonb,
  diagnosis_answers JSONB DEFAULT '[]'::jsonb,
  diagnosis_data JSONB DEFAULT '{}'::jsonb,
  analysis JSONB DEFAULT '{}'::jsonb,
  plan JSONB DEFAULT '{}'::jsonb,
  progress_pct NUMERIC(5,2) DEFAULT 0,
  next_check_at TIMESTAMPTZ,
  mentor_id TEXT DEFAULT '',
  culture_code TEXT DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ
);

CREATE INDEX idx_engagements_tenant_phase ON engagements(tenant_id, phase);
CREATE INDEX idx_engagements_next_check ON engagements(next_check_at)
  WHERE phase IN ('executing', 'tracking');

CREATE TABLE engagement_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  engagement_id UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
  action_type TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  params JSONB NOT NULL DEFAULT '{}'::jsonb,
  owner_name TEXT DEFAULT '',
  priority TEXT DEFAULT 'medium',
  due_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'pending',
  approved_at TIMESTAMPTZ,
  executed_at TIMESTAMPTZ,
  result JSONB DEFAULT '{}'::jsonb,
  linked_task_id UUID REFERENCES tasks(id),
  linked_meeting_id UUID REFERENCES meetings(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_engagement_actions_engagement ON engagement_actions(engagement_id);
CREATE INDEX idx_engagement_actions_status ON engagement_actions(engagement_id, status);
```

- [ ] **Step 2: Write migration down**

Create `sql/migrations/000019_engagements.down.sql`:

```sql
DROP TABLE IF EXISTS engagement_actions;
DROP TABLE IF EXISTS engagements;
```

- [ ] **Step 3: Write sqlc queries**

Create `sql/queries/engagements.sql`:

```sql
-- name: CreateEngagement :one
INSERT INTO engagements (tenant_id, title, problem_statement, tier, category, phase,
  diagnosis_data, mentor_id, culture_code, next_check_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetEngagement :one
SELECT * FROM engagements WHERE id = $1;

-- name: ListEngagementsByTenant :many
SELECT * FROM engagements
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveEngagements :many
SELECT * FROM engagements
WHERE tenant_id = $1 AND phase NOT IN ('closed')
ORDER BY updated_at DESC;

-- name: ListEngagementsForTracking :many
SELECT * FROM engagements
WHERE phase IN ('executing', 'tracking')
  AND (next_check_at IS NULL OR next_check_at <= now());

-- name: UpdateEngagementPhase :exec
UPDATE engagements SET phase = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementDiagnosis :exec
UPDATE engagements SET
  diagnosis_questions = $2, diagnosis_answers = $3, updated_at = now()
WHERE id = $1;

-- name: UpdateEngagementAnalysis :exec
UPDATE engagements SET analysis = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementPlan :exec
UPDATE engagements SET plan = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementProgress :exec
UPDATE engagements SET
  progress_pct = $2, next_check_at = $3, updated_at = now()
WHERE id = $1;

-- name: CloseEngagement :exec
UPDATE engagements SET phase = 'closed', closed_at = now(), updated_at = now()
WHERE id = $1;

-- name: VerifyEngagementTenant :one
SELECT tenant_id FROM engagements WHERE id = $1;

-- name: CreateEngagementAction :one
INSERT INTO engagement_actions (engagement_id, action_type, title, description,
  params, owner_name, priority, due_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListEngagementActions :many
SELECT * FROM engagement_actions
WHERE engagement_id = $1
ORDER BY priority_order(priority), created_at;

-- name: UpdateEngagementActionStatus :exec
UPDATE engagement_actions SET status = $2, updated_at = now() WHERE id = $1;

-- name: ApproveEngagementAction :exec
UPDATE engagement_actions SET status = 'approved', approved_at = now(), updated_at = now()
WHERE id = $1;

-- name: RejectEngagementAction :exec
UPDATE engagement_actions SET status = 'rejected', updated_at = now() WHERE id = $1;

-- name: MarkEngagementActionDone :exec
UPDATE engagement_actions SET
  status = 'done', executed_at = now(), result = $2, updated_at = now()
WHERE id = $1;

-- name: MarkEngagementActionFailed :exec
UPDATE engagement_actions SET
  status = 'failed', result = $2, updated_at = now()
WHERE id = $1;

-- name: LinkEngagementActionTask :exec
UPDATE engagement_actions SET linked_task_id = $2, updated_at = now() WHERE id = $1;

-- name: LinkEngagementActionMeeting :exec
UPDATE engagement_actions SET linked_meeting_id = $2, updated_at = now() WHERE id = $1;

-- name: CountEngagementActionsByStatus :many
SELECT status, COUNT(*)::int AS count
FROM engagement_actions WHERE engagement_id = $1
GROUP BY status;
```

Note: The `priority_order(priority)` function may not exist. Use a CASE expression instead:
```sql
ORDER BY CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END, created_at;
```

- [ ] **Step 4: Run sqlc generate**

```bash
cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate
```

Verify no errors. Check that `internal/db/sqlc/engagements.sql.go` is generated.

- [ ] **Step 5: Verify build**

```bash
cd /Users/anna/Documents/ai-management-brain && go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add sql/migrations/000019_* sql/queries/engagements.sql internal/db/sqlc/
git commit -m "feat: add engagements migration and sqlc queries"
```

---

### Task 2: Fix Dispatcher Stubs

**Files:**
- Modify: `internal/brain/dispatcher.go`

The Dispatcher has 3 methods that only log but don't write to DB. Fix them to actually create records.

- [ ] **Step 1: Read existing dispatcher code**

Read `internal/brain/dispatcher.go` to understand current signatures and the `dispatchParseUUID` helper.

- [ ] **Step 2: Extend ActionResult**

Add optional ID fields to ActionResult:

```go
type ActionResult struct {
    Index             int    `json:"index"`
    Success           bool   `json:"success"`
    Message           string `json:"message,omitempty"`
    Error             string `json:"error,omitempty"`
    Skipped           string `json:"skipped,omitempty"`
    NeedsConfirmation bool   `json:"needs_confirmation,omitempty"`
    Link              string `json:"link,omitempty"`
    TaskID            string `json:"task_id,omitempty"`
    MeetingID         string `json:"meeting_id,omitempty"`
    SignalID          string `json:"signal_id,omitempty"`
}
```

- [ ] **Step 3: Fix createTask**

Replace the stub with actual DB write:

```go
func (d *Dispatcher) createTask(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
    title, _ := params["title"].(string)
    if title == "" {
        return ActionResult{Error: "missing task title"}
    }
    description, _ := params["description"].(string)
    priority, _ := params["priority"].(string)
    if priority == "" {
        priority = "medium"
    }

    // Resolve owner_id if provided (by name or UUID)
    var ownerID pgtype.UUID
    if ownerStr, ok := params["owner_id"].(string); ok && ownerStr != "" {
        ownerID, _ = dispatchParseUUID(ownerStr)
    }

    // Parse due_at if provided
    var dueAt pgtype.Timestamptz
    if dueStr, ok := params["due_at"].(string); ok && dueStr != "" {
        if t, err := time.Parse(time.RFC3339, dueStr); err == nil {
            dueAt = pgtype.Timestamptz{Time: t, Valid: true}
        }
    }

    task, err := d.queries.CreateTask(ctx, sqlc.CreateTaskParams{
        TenantID:       tenantID,
        Title:          title,
        Description:    description,
        OwnerID:        ownerID,
        Status:         "todo",
        Priority:       priority,
        DueAt:          dueAt,
        SourceSystem:   pgtype.Text{String: "consulting", Valid: true},
        CreatedByAgent: true,
    })
    if err != nil {
        slog.Error("dispatcher: create_task failed", "error", err)
        return ActionResult{Error: fmt.Sprintf("create task failed: %v", err)}
    }
    return ActionResult{
        Success: true,
        Message: fmt.Sprintf("Task created: %s", title),
        TaskID:  task.ID.String(),
    }
}
```

Note: Check `sqlc.CreateTaskParams` field names match generated code. May need `pgtype.Text` for nullable strings.

- [ ] **Step 4: Fix scheduleMeeting**

```go
func (d *Dispatcher) scheduleMeeting(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
    empIDStr, _ := params["employee_id"].(string)
    if empIDStr == "" {
        return ActionResult{Error: "missing employee_id"}
    }
    empID, err := dispatchParseUUID(empIDStr)
    if err != nil {
        return ActionResult{Error: "invalid employee_id"}
    }

    notes, _ := params["agenda"].(string)
    if notes == "" {
        notes, _ = params["notes"].(string)
    }
    durationMin := int32(30) // default
    if d, ok := params["duration_min"].(float64); ok {
        durationMin = int32(d)
    }

    meeting, err := d.queries.CreateMeeting(ctx, sqlc.CreateMeetingParams{
        TenantID:    tenantID,
        EmployeeID:  empID,
        MeetingDate: pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
        DurationMin: pgtype.Int4{Int32: durationMin, Valid: true},
        Notes:       pgtype.Text{String: notes, Valid: notes != ""},
        Mood:        pgtype.Text{String: "neutral", Valid: true},
    })
    if err != nil {
        slog.Error("dispatcher: schedule_meeting failed", "error", err)
        return ActionResult{Error: fmt.Sprintf("create meeting failed: %v", err)}
    }
    return ActionResult{
        Success:   true,
        Message:   fmt.Sprintf("1:1 meeting scheduled with employee %s", empIDStr[:min(len(empIDStr), 8)]),
        MeetingID: meeting.ID.String(),
    }
}
```

Note: Check exact `sqlc.CreateMeetingParams` fields. The `meeting_date` param might need a specific format. The `manager_id` field may be required — check the meetings table schema.

- [ ] **Step 5: Fix flagRisk**

```go
func (d *Dispatcher) flagRisk(ctx context.Context, tenantID pgtype.UUID, params map[string]any) ActionResult {
    riskDesc, _ := params["risk_description"].(string)
    if riskDesc == "" {
        return ActionResult{Error: "missing risk_description"}
    }

    subjectType, _ := params["subject_type"].(string)
    if subjectType == "" {
        subjectType = "team"
    }
    var subjectID pgtype.UUID
    if sid, ok := params["subject_id"].(string); ok && sid != "" {
        subjectID, _ = dispatchParseUUID(sid)
    }
    severity := float64(7.0)
    if s, ok := params["severity"].(float64); ok {
        severity = s
    }

    signal, err := d.queries.CreateExecutionSignal(ctx, sqlc.CreateExecutionSignalParams{
        TenantID:    tenantID,
        SubjectType: subjectType,
        SubjectID:   subjectID,
        SignalType:  "risk_flag",
        Score:       pgtype.Numeric{/* severity */},
        Reasons:     []byte(fmt.Sprintf(`["%s"]`, riskDesc)),
    })
    // ... handle error, return result
}
```

Note: Check if `CreateExecutionSignal` query exists in `sql/queries/execution_signals.sql`. If not, add it. The `Score` field uses `pgtype.Numeric` which needs careful handling — check how other code handles it (e.g., incentive_scores). If too complex, fall back to creating a `communication_event` with type `risk_flagged` instead.

- [ ] **Step 6: Add `time` import**

Add `"time"` to the import block in dispatcher.go.

- [ ] **Step 7: Verify build**

```bash
go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add internal/brain/dispatcher.go
git commit -m "fix: wire dispatcher stubs to actually write tasks, meetings, signals to DB"
```

---

### Task 3: Consulting Engine Core

**Files:**
- Create: `internal/brain/consulting_prompts.go`
- Create: `internal/brain/consulting.go`

- [ ] **Step 1: Write consulting_prompts.go**

Create `internal/brain/consulting_prompts.go` with LLM prompts for each phase:

```go
package brain

const classifyEngagementPrompt = `You are a management consultant AI. Classify this management problem.

Problem: %s

Available system data summary:
%s

Respond in JSON:
{
  "tier": "quick|standard|deep",
  "category": "people|process|strategy|performance|organization",
  "title": "Short descriptive title (max 50 chars)",
  "reasoning": "One sentence explaining classification",
  "first_question": "Your first diagnostic question to the boss"
}

Classification rules:
- quick: Single person or clear issue, strong data signals, 1-3 actions needed
- standard: Multi-factor, needs context gathering, 3-7 actions
- deep: Organizational/structural/cultural, sparse data, 5-15 actions

Return ONLY valid JSON.`

const diagnosisQuestionPrompt = `You are a management consultant conducting a structured diagnosis.

Problem: %s
Tier: %s (max %d questions remaining)
Category: %s

System data:
%s

Conversation so far:
%s

Generate the next diagnostic question. Ask ONE focused question that will help identify root causes.

Respond in JSON:
{
  "question": "Your next question",
  "reasoning": "Why this question matters (internal, not shown to boss)",
  "sufficient": false
}

Set "sufficient": true if you have enough information to proceed to analysis.
When sufficient, still provide a question field (it will be ignored).

Key principles:
- Ask about CAUSES, not symptoms
- Reference specific data points when relevant
- Don't repeat information already gathered
- If data strongly suggests root cause, mark sufficient early

Return ONLY valid JSON.`

const analysisPrompt = `You are a senior management consultant producing a root cause analysis.

Problem: %s
Category: %s

Diagnosis conversation:
%s

System data:
%s

Past strategy memories (what worked/didn't before):
%s

Produce a structured analysis. Respond in JSON:
{
  "root_causes": [
    {"cause": "description", "confidence": 0.9, "evidence": ["data point 1", "boss said X"]}
  ],
  "frameworks_applied": ["Framework Name: finding"],
  "key_insights": ["insight 1", "insight 2"],
  "risk_factors": ["risk 1"]
}

Apply relevant frameworks:
- People issues → Herzberg motivation theory, Situational Leadership
- Process issues → Theory of Constraints, Lean principles
- Strategy issues → Porter's competitive forces, SWOT
- Performance issues → OKR cascade analysis, KPI alignment
- Organization issues → McKinsey 7S, Org design principles

Return ONLY valid JSON.`

const planGenerationPrompt = `You are a management consultant generating an actionable management plan.

Problem: %s
Analysis:
%s

Team members: %s

Available action types and their params:
- create_task: {title, description, owner_id, owner_name, priority, due_at}
- schedule_meeting: {employee_id, employee_name, agenda, duration_min}
- send_message: {employee_id, employee_name, message}
- flag_risk: {risk_description, subject_type, subject_id, severity}
- monitor: {description, check_frequency}
- follow_up: {employee_id, employee_name, topic, days_from_now}

Respond in JSON:
{
  "summary": "2-3 sentence plan summary",
  "expected_outcomes": ["outcome 1", "outcome 2"],
  "timeline": "e.g., 2 weeks",
  "actions": [
    {
      "action_type": "create_task",
      "title": "Specific action",
      "description": "Details",
      "params": {"title": "...", "owner_name": "...", "priority": "high", "due_at": "2026-04-03"},
      "owner_name": "person name",
      "priority": "high",
      "reason": "Why this action, linked to root cause"
    }
  ]
}

Rules:
- Each action MUST have a specific owner (use actual team member names)
- Each action MUST be measurable (can verify completion)
- Prioritize: critical > high > medium > low
- For quick tier: max 3 actions. Standard: max 7. Deep: max 15.
- Use send_message for immediate communication needs
- Use schedule_meeting for 1:1s that need to happen
- Use create_task for trackable deliverables
- Use monitor for ongoing observation without immediate action

Return ONLY valid JSON.`

const progressReportPrompt = `You are tracking a management consulting engagement.

Original problem: %s
Plan summary: %s

Action status:
%s

Generate a brief progress update for the boss. Be specific about what's done, what's pending, and flag anything concerning.

Keep it under 200 words. Use bullet points. Be direct.`

const closeSummaryPrompt = `You are closing a management consulting engagement.

Problem: %s
Plan: %s
Action outcomes: %s
Duration: %s

Generate:
1. A brief effectiveness summary (what worked, what didn't)
2. Key lessons learned (for future similar situations)
3. An effectiveness score (1-10)

Respond in JSON:
{
  "summary": "2-3 sentences",
  "lessons": ["lesson 1", "lesson 2"],
  "effectiveness_score": 8,
  "what_worked": ["action X"],
  "what_didnt": ["action Y"]
}

Return ONLY valid JSON.`
```

- [ ] **Step 2: Write consulting.go**

Create `internal/brain/consulting.go`:

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
    "github.com/tonypk/ai-management-brain/internal/memory"
)

type ConsultingEngine struct {
    llm            LLMClient
    contextService *ContextService
    dispatcher     *Dispatcher
    queries        *sqlc.Queries
    memStore       *memory.MemoryStore
}

func NewConsultingEngine(
    llm LLMClient,
    cs *ContextService,
    dispatcher *Dispatcher,
    queries *sqlc.Queries,
    memStore *memory.MemoryStore,
) *ConsultingEngine {
    return &ConsultingEngine{
        llm: llm, contextService: cs, dispatcher: dispatcher,
        queries: queries, memStore: memStore,
    }
}

// tierMaxQuestions returns max diagnosis questions per tier.
func tierMaxQuestions(tier string) int {
    switch tier {
    case "quick": return 2
    case "deep": return 10
    default: return 5 // standard
    }
}
```

Then implement:

**StartEngagement**: Pull company context → call LLM with classifyEngagementPrompt → parse tier/category/title/first_question → create engagement record → return first question.

**AnswerQuestion**: Load engagement → append answer to diagnosis_answers → call LLM with diagnosisQuestionPrompt → if sufficient OR max questions reached: run analysis (analysisPrompt) → generate plan (planGenerationPrompt) → create engagement_actions → update phase to "plan" → return plan. Otherwise: return next question.

**ReviewAction**: Load action → set approved/rejected → if all actions reviewed: update engagement phase to "review" complete.

**ExecuteApproved**: Load all approved actions → for each: call dispatcher.Execute → store result → link task/meeting IDs → update phase to "executing"/"tracking" → set next_check_at.

**CheckProgress**: Load engagement + actions → check linked task statuses → calculate progress → format report via LLM → update progress_pct → return report.

**CloseEngagement**: Call closeSummaryPrompt → store lessons as strategy_result memories → mark engagement closed.

Each method should be 30-60 lines. Total file ~400 lines.

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/brain/consulting_prompts.go internal/brain/consulting.go
git commit -m "feat: add consulting engine core with 6 lifecycle methods"
```

---

### Task 4: Consulting Engine Tests

**Files:**
- Create: `internal/brain/consulting_test.go`

- [ ] **Step 1: Write unit tests**

Test key functions:
- `TestTierMaxQuestions` — verify quick=2, standard=5, deep=10
- `TestClassifyResponse parsing` — verify JSON parsing of tier classification
- `TestDiagnosisQuestionParsing` — verify question + sufficient flag parsing
- `TestAnalysisParsing` — verify root cause JSON parsing
- `TestPlanParsing` — verify action list parsing
- `TestProgressCalculation` — verify progress percentage from action statuses

Use mock LLM client (check existing tests for pattern — likely `internal/brain/recommender_test.go` or similar).

- [ ] **Step 2: Run tests**

```bash
go test ./internal/brain/ -v -run TestConsulting
```

- [ ] **Step 3: Commit**

```bash
git add internal/brain/consulting_test.go
git commit -m "test: add consulting engine unit tests"
```

---

### Task 5: Bot Integration — /consult Command

**Files:**
- Modify: `internal/brain/orchestrator.go` (add IntentConsult)
- Modify: `internal/bot/commands.go` (add /consult + routing)

- [ ] **Step 1: Add IntentConsult to orchestrator**

In `internal/brain/orchestrator.go`, add:

```go
const IntentConsult IntentType = "consult"
```

Add consulting pattern matching in `matchPattern`:
```go
// "/consult" or "/consult status" or "/consult close"
if strings.HasPrefix(lower, "/consult") || strings.HasPrefix(lower, "consult") {
    rest := strings.TrimPrefix(strings.TrimPrefix(lower, "/"), "consult")
    rest = strings.TrimSpace(rest)
    return Intent{Type: IntentConsult, Content: rest, OriginalNL: original}, true
}
```

Add consulting intent to LLM classifier prompt:
```
- consult: Boss describes a management problem/challenge and wants structured help
```

- [ ] **Step 2: Add ConsultingHandler interface to bot**

In `internal/bot/commands.go`, add interface and field:

```go
type ConsultingHandler interface {
    StartEngagement(ctx context.Context, tenantID string, problem, mentorID, cultureCode string) (engagementID, firstQuestion string, err error)
    AnswerQuestion(ctx context.Context, engagementID, answer string) (nextQuestion string, planText string, done bool, err error)
    ReviewActions(ctx context.Context, engagementID string, approvals map[string]bool) error
    ExecuteApproved(ctx context.Context, engagementID string) (string, error)
    GetStatus(ctx context.Context, engagementID string) (string, error)
    CloseEngagement(ctx context.Context, engagementID string) (string, error)
}
```

Add to CommandHandler:
```go
consulting ConsultingHandler
```

Add setter:
```go
func (h *CommandHandler) SetConsulting(c ConsultingHandler) { h.consulting = c }
```

- [ ] **Step 3: Implement HandleConsult**

```go
func (h *CommandHandler) HandleConsult(c BotContext) error {
    // Parse subcommand: status, close, history, or start new
    // Use Redis for active engagement state:
    //   key: consulting:{tenant_id}:active → engagement_id
    //   key: consulting:{tenant_id}:phase → current phase
    // Route to appropriate method
}
```

- [ ] **Step 4: Add consulting message routing in main bot handler**

When boss sends a non-command message and active engagement exists in Redis:
- If phase=diagnosis → route to AnswerQuestion
- If phase=review → parse approval input (e.g., "1,3,5 yes" or "all yes")
- Otherwise → normal flow

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/brain/orchestrator.go internal/bot/commands.go
git commit -m "feat: add /consult bot command with engagement routing"
```

---

### Task 6: Wiring in main.go + API Routes

**Files:**
- Modify: `cmd/brain/main.go`
- Modify: `internal/api/router.go`
- Create: `internal/api/engagement_handlers.go`

- [ ] **Step 1: Create engagement API handlers**

Create `internal/api/engagement_handlers.go` with REST handlers:
- `GET /api/v1/engagements` — list engagements
- `GET /api/v1/engagements/:id` — get engagement detail
- `POST /api/v1/engagements` — start engagement (body: {problem_statement})
- `POST /api/v1/engagements/:id/answer` — answer diagnosis question
- `POST /api/v1/engagements/:id/actions/:action_id/approve` — approve action
- `POST /api/v1/engagements/:id/actions/:action_id/reject` — reject action
- `POST /api/v1/engagements/:id/execute` — execute approved actions
- `POST /api/v1/engagements/:id/close` — close engagement

- [ ] **Step 2: Add routes to router.go**

Add `ConsultingEngine *brain.ConsultingEngine` to RouterConfig.
Register routes under boss-only group.

- [ ] **Step 3: Wire in main.go**

After existing engine creation:
```go
consultingEngine := brain.NewConsultingEngine(llmSvc, contextService, dispatcher, queries, memStore)
```

Add to RouterConfig:
```go
ConsultingEngine: consultingEngine,
```

Wire to bot:
```go
cmdHandler.SetConsulting(consultingBotAdapter)
```

Add scheduler job for engagement tracking.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/engagement_handlers.go internal/api/router.go cmd/brain/main.go
git commit -m "feat: wire consulting engine into main, API routes, and scheduler"
```

---

### Task 7: Engagement Tracker Scheduler Job

**Files:**
- Create: `internal/brain/engagement_tracker.go`

- [ ] **Step 1: Write tracker**

```go
package brain

// EngagementTracker checks progress on active engagements and pushes updates.
type EngagementTracker struct {
    consulting *ConsultingEngine
    sender     channel.Sender
    queries    *sqlc.Queries
}

// Run is called by the scheduler daily.
func (t *EngagementTracker) Run(ctx context.Context) error {
    // 1. List engagements due for checking
    // 2. For each: call consulting.CheckProgress
    // 3. Send progress to boss via sender
    // 4. If all done → suggest closing
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/brain/engagement_tracker.go
git commit -m "feat: add engagement tracker scheduler job for daily progress push"
```

---

### Task 8: MCP Tools

**Files:**
- Modify: `mcp-server/src/server.ts`

- [ ] **Step 1: Add 5 new MCP tools**

Add to the MCP server:

```typescript
// Read tools
server.tool("list_engagements", "List consulting engagements with status and progress", {...})
server.tool("get_engagement_status", "Get detailed status of a consulting engagement", {...})

// Write tools
server.tool("start_engagement", "Start a new consulting engagement from a problem description", {...})
server.tool("answer_diagnosis", "Answer a diagnosis question in an active engagement", {...})
server.tool("review_engagement_actions", "Approve or reject actions in a consulting engagement", {...})
```

Each tool calls the corresponding `/api/v1/openclaw/engagements/...` endpoint.

- [ ] **Step 2: Add openclaw API routes**

Add to router.go under openclaw group:
```go
oc.GET("/engagements", ...)
oc.GET("/engagements/:id", ...)
oc.POST("/engagements", ...)
oc.POST("/engagements/:id/answer", ...)
oc.POST("/engagements/:id/actions/review", ...)
```

- [ ] **Step 3: Verify MCP server builds**

```bash
cd mcp-server && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add mcp-server/src/server.ts internal/api/router.go
git commit -m "feat: add 5 MCP tools for consulting engagements"
```

---

### Task 9: Integration Test + Deploy

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/anna/Documents/ai-management-brain && go test ./...
```

- [ ] **Step 2: Run migration on server**

```bash
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml exec postgres psql -U brain -d brain -f /docker-entrypoint-initdb.d/000019_engagements.up.sql'
```

Or apply via the app's migration runner if available.

- [ ] **Step 3: Build and deploy**

```bash
# Cross-compile
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/brain-binary ./cmd/brain

# SCP to server
scp /tmp/brain-binary ai-brain:~/ai-management-brain/brain

# Deploy
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build brain'

# Build and deploy MCP
cd mcp-server && npm run build
scp -r dist/ ai-brain:~/ai-management-brain/mcp-server/dist/
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build mcp'
```

- [ ] **Step 4: Health check**

```bash
ssh ai-brain 'curl -s localhost/healthz'
```

- [ ] **Step 5: Test via Telegram**

Send `/consult 团队效率下降了` to the bot. Verify:
1. AI classifies and returns first question
2. Answer the question → get next question
3. After diagnosis → get plan with action items
4. Approve actions → they execute (check DB for new tasks/meetings)

- [ ] **Step 6: Update README/SKILL version**

Bump version to 7.0.0 in `openclaw-skill/README.md` — this is a major feature.

- [ ] **Step 7: Final commit and push**

```bash
git add -A
git commit -m "feat: consulting engine v7.0 — AI McKinsey diagnosis + execution + tracking"
git push origin main
```
