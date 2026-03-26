# AI Recommendation Engine — Design Spec

## Goal

Add a proactive AI recommendation engine that analyzes team data, generates actionable management suggestions, and supports one-click execution. Delivered via Web dashboard, dedicated page, Telegram push, and MCP tools.

## Architecture

Two-pipeline hybrid: daily batch analysis (cron, 10:30 AM) for cross-person/cross-project trend detection, plus real-time event-driven triggers for urgent situations. Both pipelines write to a shared `recommendations` table. High-priority recommendations push to Telegram; all are visible on web.

## Tech Stack

- Go (Gin + sqlc + pgx) backend
- Vue3 + TypeScript + NaiveUI frontend
- PostgreSQL 16 (new `recommendations` table)
- Claude Sonnet 4 API (daily scan + complex realtime)
- Go templates (simple realtime triggers, zero API cost)
- Telegram Bot (push + inline commands)
- MCP Streamable HTTP (2 new tools via existing Node.js MCP server)

---

## Section 1: Data Model

### recommendations table

```sql
CREATE TABLE recommendations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    category        TEXT NOT NULL CHECK (category IN ('people', 'project', 'kpi', 'organization')),
    priority        TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    suggested_actions JSONB NOT NULL DEFAULT '[]',
    evidence        JSONB NOT NULL DEFAULT '{}',
    source          TEXT NOT NULL CHECK (source IN ('daily_scan', 'realtime_trigger')),
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'dismissed', 'executed', 'expired')),
    target_entity_type TEXT CHECK (target_entity_type IN ('employee', 'project', 'metric', 'goal') OR target_entity_type IS NULL),
    target_entity_id   UUID, -- FK to the relevant entity, NULL for org-level recommendations
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at     TIMESTAMPTZ,
    executed_at     TIMESTAMPTZ
);

CREATE INDEX idx_recommendations_tenant_status ON recommendations(tenant_id, status);
CREATE INDEX idx_recommendations_tenant_created ON recommendations(tenant_id, created_at DESC);
CREATE INDEX idx_recommendations_expires ON recommendations(tenant_id, expires_at) WHERE status = 'pending';
```

### Suggested Action — Full Params Schema Per Type

```jsonc
// schedule_meeting
{
  "type": "schedule_meeting",
  "params": {
    "employee_id": "uuid",
    "meeting_type": "one_on_one",           // one_on_one | team
    "suggested_date": "2026-03-28",         // optional, YYYY-MM-DD
    "notes": "Follow up on sentiment drop"  // optional
  },
  "label": "安排 1:1 会议"
}

// send_message
{
  "type": "send_message",
  "params": {
    "employee_id": "uuid",
    "message": "最近还好吗？有什么需要帮助的吗？"
    // channel auto-resolved via employee's preferred channel
  },
  "label": "发送关怀消息"
}

// create_task
{
  "type": "create_task",
  "params": {
    "title": "Profile database queries under load",
    "assignee_id": "uuid",                  // employee UUID
    "project_id": "uuid",                   // optional
    "priority": "high",                     // low | medium | high | critical
    "due_at": "2026-03-31T00:00:00Z"        // optional
  },
  "label": "创建任务"
}

// reassign_task (requires confirmation)
{
  "type": "reassign_task",
  "params": {
    "task_id": "uuid",
    "new_assignee_id": "uuid",
    "reason": "Current assignee overloaded"
  },
  "label": "重新分配任务"
}

// flag_risk
{
  "type": "flag_risk",
  "params": {
    "project_id": "uuid",
    "risk_description": "Backend API performance bottleneck"
  },
  "label": "标记项目风险"
}

// adjust_target (link only, no auto-execute)
{
  "type": "adjust_target",
  "params": {
    "entity_type": "metric",                // metric | goal
    "entity_id": "uuid",
    "suggested_value": 80000,               // optional hint
    "link": "/metrics"                      // deeplink to edit page
  },
  "label": "查看并调整目标"
}

// public_recognition
{
  "type": "public_recognition",
  "params": {
    "employee_id": "uuid",
    "message": "Alice completed the payment gateway 2 days ahead of schedule!"
  },
  "label": "公开表扬"
}

// create_suggestion
{
  "type": "create_suggestion",
  "params": {
    "title": "增加 code reviewer 人数",
    "content": "代码审查成为瓶颈，建议从 2 人增加到 3 人",
    "capability": "org_optimization"
  },
  "label": "创建组织优化建议"
}
```

### Evidence JSON Schema

```jsonc
{
  "signals": [
    {"name": "engagement_drop", "value": 0.72},
    {"name": "delivery_risk", "value": 0.65}
  ],
  "metrics": [
    {"name": "revenue", "trend": "declining_2w"}
  ],
  "employees": [
    {"name": "Alice", "issue": "sentiment_drop_3d"}
  ],
  "tasks": [
    {"id": "task-123", "issue": "overdue_5d"}
  ]
}
```

### Supported Action Types

| Type | Effect | Auto-execute | Category |
|------|--------|-------------|----------|
| `schedule_meeting` | Create 1:1 meeting record | Yes | people |
| `send_message` | Send via employee's preferred channel | Yes | people |
| `create_task` | Create and assign task | Yes | project |
| `reassign_task` | Reassign existing task | Confirm required | project |
| `flag_risk` | Update project risk status | Yes | project |
| `adjust_target` | Navigate to edit page (no auto-execute) | No — link only | kpi |
| `public_recognition` | Send recognition to group | Yes | people |
| `create_suggestion` | Create org suggestion record | Yes | organization |

---

## Section 2: Analysis Pipelines

### Pipeline 1: Daily Batch Scan

- **Schedule**: `30 10 * * *` (10:30 AM daily, after check-ins, before chase)
- **Registered as**: `recommendation_scan` cron job via `sched.AddJob()`

**Data inputs (8 sources):**

1. Execution signals (last 7 days)
2. Communication events (last 7 days)
3. Overdue tasks + blocked projects
4. Metric values trend (last 30 days)
5. Goal snapshot deviations (current cycle)
6. Employee attendance/sentiment trends (last 14 days)
7. Pending recommendations (for dedup)
8. Latest working memory snapshot

**Input truncation strategy:**
- Each data source is summarized/truncated to fit within a token budget (~2000 tokens per source, ~16000 total input budget)
- Signals: top 10 by score
- Events: aggregated counts, not raw events
- Metrics: only metrics with deviation > 10% from target
- Employees: only those with anomalies (missed days, sentiment drops, overload)
- If total input exceeds budget, lower-priority sources are truncated first (memory → events → signals)

**Process:**
1. Gather all 8 data sources, apply truncation
2. Single Claude API call via `llm.ChatLong()` with mentor-aware system prompt
3. Parse JSON array output (max 5 recommendations)
4. Dedup against pending recommendations (see Dedup Strategy below)
5. Store in `recommendations` table with `source = 'daily_scan'`, `expires_at = now() + 72h`
6. Push critical/high priority via Telegram

**LLM failure handling:** If Claude API call fails (timeout, rate limit, auth error), log the error and skip this scan. The next daily scan will catch up. No partial recommendations are stored.

### Pipeline 2: Real-time Event Triggers

**Integration points** (3 call sites, not just triggers.go):

1. **After report submission** (`internal/report/summarizer.go`): Check per-employee triggers (consecutive_miss, sentiment_drop, performance_spike)
2. **After signal generation** (`internal/brain/state_engine.go`): Check signal-based triggers (signal_high_score, blocker_cascade)
3. **After metric value update** (`internal/api/metric_handlers.go`): Check metric_anomaly trigger

Each call site invokes `recommender.RealtimeEvaluate(ctx, event)` with the relevant event data.

**Trigger rules:**

| Event | Condition | Template or LLM | Call Site |
|-------|-----------|-----------------|-----------|
| `consecutive_miss` | 3+ days missed | Template | report submission |
| `sentiment_drop` | 3 consecutive negative | Template | report submission |
| `signal_high_score` | Any signal >= 0.7 | LLM (needs context) | signal generation |
| `task_overdue_critical` | Critical task 3+ days overdue | Template | daily scan only |
| `blocker_cascade` | 3+ people blocked by same issue | LLM | signal generation |
| `performance_spike` | Exceptional delivery streak | Template | report submission |
| `metric_anomaly` | KPI off-target 2+ periods | LLM | metric value update |

**Process:**
1. Trigger rule matches (Go code, no LLM)
2. Check dedup (see Dedup Strategy below)
3. Generate recommendation: template (simple) or lightweight LLM call (complex)
4. Store with `source = 'realtime_trigger'`, `expires_at = now() + 72h`
5. If priority = critical, push immediately via Telegram

### Dedup Strategy

- Match on: `category` + `target_entity_type` + `target_entity_id` + status = pending + created within 72h
- **For org-level recommendations** (target_entity_id IS NULL): match on `category` + `title similarity` (exact match) instead
- If new recommendation has higher priority → expire old one, create new
- Dedup window: 72h (hardcoded; sufficient for daily scan cadence)

### Cost Control

- Daily scan: 1 Claude API call per tenant per day
- Realtime: simple triggers use Go templates (0 API calls); complex triggers use 1 short LLM call
- Estimated: 1-3 Claude calls per tenant per day

---

## Section 3: Delivery Channels

### Push Rules by Priority

| Priority | Web Dashboard | Telegram | MCP |
|----------|--------------|----------|-----|
| critical | Red banner + notification | Immediate push | Returned by `get_recommendations` |
| high | Orange card | Bundled with daily summary | Same |
| medium | Recommendations page only | Not pushed | Same |
| low | Recommendations page only | Not pushed | Same |

### Telegram Format & Tenant Isolation

Telegram commands resolve tenant via the existing `bot.ResolveTenantByChat(chatID)` lookup (same as `/status`, `/mentor` commands). The bot handler:
1. Resolves tenant_id from Telegram chat_id via `channels` table
2. Queries recommendations filtered by that tenant_id
3. Execute/dismiss verifies the recommendation belongs to the resolved tenant

```
🧠 AI Management Insight

📋 {title}
{description}

👉 /execute_rec_{short_id}  — {primary_action_label}
❌ /dismiss_rec_{short_id}  — Ignore
```

### Web: Dashboard Section

New section below alerts on Dashboard page:

- Shows top 3 pending recommendations (highest priority first)
- Each card: priority icon + title + [Execute] [Dismiss] buttons
- "View All →" link to /recommendations page

### Web: Recommendations Page (`/recommendations`)

- Route: `/recommendations`
- Sidebar: Organize group, after Dashboard, with pending count badge
- Tabs: Pending | Executed | Dismissed
- Each recommendation card shows:
  - Priority badge (color-coded)
  - Title + description
  - Evidence tags (signals, metrics, employees)
  - Source label (daily_scan / realtime_trigger) + timestamp
  - Action buttons per suggested_action
  - [Execute All] [Dismiss] footer buttons

### MCP Tools (2 new)

Added to the existing Node.js MCP server (`mcp/src/`), calling the Go API endpoints:

```
get_recommendations      — GET /api/v1/recommendations?status=pending → List pending
execute_recommendation   — POST /api/v1/recommendations/:id/execute → Execute action
```

The MCP server authenticates to the Go API using the tenant's API key (same pattern as existing MCP tools).

---

## Section 4: Action Dispatcher

### Location: `internal/brain/dispatcher.go`

### Single Action Flow

```
POST /recommendations/:id/execute { action_index: 0 }
  → Load recommendation + verify tenant
  → Extract action by index
  → If action requires confirmation (reassign_task): return { needs_confirmation: true, ... }
  → If action is link-only (adjust_target): return { link: "/metrics/..." }
  → Otherwise: dispatcher.Execute(ctx, action)
  → On success: if all actions executed → status = "executed", else stays "pending"
  → Return result
```

### Execute-All Flow

```
POST /recommendations/:id/execute-all
  → Load recommendation + verify tenant
  → For each action in suggested_actions:
      → Skip actions that require confirmation (reassign_task)
      → Skip link-only actions (adjust_target)
      → Execute auto-executable actions
  → Return results array: [{ index: 0, success: true }, { index: 1, skipped: "requires_confirmation" }, ...]
  → If all executable actions succeeded → status = "executed"
  → If some failed → status stays "pending", return partial results
```

**Partial failure**: best-effort execution. Each action is independent. Failed actions are reported in the response but don't block others. The recommendation stays `pending` until all executable actions succeed (or boss dismisses it).

### Dispatch Table

```go
switch action.Type {
case "schedule_meeting":   → meetingQueries.CreateMeeting(tenantID, params)
case "send_message":       → channel.ResolveChannel(employeeID) → sender.Send(msg)
case "create_task":        → taskQueries.CreateTask(tenantID, params)
case "reassign_task":      → return NeedsConfirmation (Web UI confirms, then re-calls with confirmed=true)
case "flag_risk":          → projectQueries.UpdateProject(projectID, risk_flags)
case "adjust_target":      → return DeepLink (no execution)
case "public_recognition": → channel.ResolveBossGroup(tenantID) → sender.Send(msg)
case "create_suggestion":  → orgQueries.CreateSuggestion(tenantID, params)
}
```

### Safety

- `reassign_task`: Web UI shows confirmation dialog before executing; Telegram skips this action with note "请在 Web 端确认"
- `adjust_target`: Never auto-executes; returns a deeplink to the KPI/OKR edit page
- All other actions: Execute immediately on button click
- Failed execution: recommendation stays `pending`, error message returned to UI

### Result Format

```jsonc
// Single action success
{ "success": true, "message": "1:1 meeting scheduled for 2026-03-28" }

// Single action failure
{ "success": false, "error": "Employee has no active channel" }

// Execute-all result
{
  "results": [
    { "index": 0, "success": true, "message": "Message sent" },
    { "index": 1, "skipped": "requires_confirmation" }
  ],
  "all_done": false
}
```

---

## Section 5: Backend Modules & API

### New Files (6 backend + 4 frontend = 10)

**Backend:**
```
sql/migrations/000017_recommendations.up.sql      — Create table + indexes
sql/migrations/000017_recommendations.down.sql     — Drop table
sql/queries/recommendations.sql                    — sqlc queries (8 queries)
internal/brain/recommender.go                      — DailyScan() + RealtimeEvaluate()
internal/brain/dispatcher.go                       — Action execution dispatcher
internal/api/recommendation_handlers.go            — 6 HTTP handlers
```

**Frontend:**
```
frontend/src/types/recommendation.ts               — TypeScript types
frontend/src/api/recommendations.ts                — API client (6 functions)
frontend/src/views/RecommendationsView.vue         — Full page with tabs
frontend/src/components/recommendations/
  RecommendationCard.vue                           — Single card + actions
  RecommendationSummary.vue                        — Dashboard embed (top 3)
```

### Modified Files (8)

```
internal/api/router.go               — Add recommendations route group
internal/report/triggers.go          — Add realtime trigger call sites
internal/brain/state_engine.go       — Call RealtimeEvaluate after signal generation
internal/api/metric_handlers.go      — Call RealtimeEvaluate after metric update
cmd/brain/main.go                    — Register cron job + init recommender + inline migration 000017
mcp/src/tools/                       — Add 2 new MCP tool definitions
frontend/src/router/index.ts         — Add /recommendations route
frontend/src/layouts/AppLayout.vue   — Sidebar menu item + pending badge
frontend/src/views/DashboardView.vue — Embed RecommendationSummary component
```

### API Endpoints (6)

```
GET    /api/v1/recommendations              — List (filter: ?status=pending&category=people)
GET    /api/v1/recommendations/summary      — Dashboard: pending count + top 3
POST   /api/v1/recommendations/:id/execute  — Execute one action (body: {action_index})
POST   /api/v1/recommendations/:id/execute-all — Execute all auto-executable actions
POST   /api/v1/recommendations/:id/dismiss  — Mark dismissed
DELETE /api/v1/recommendations/:id          — Delete (only dismissed/expired)
```

All endpoints require authentication + `RequireRole("boss")`.

### Telegram Commands (2)

```
/recs          — List pending recommendations
/execute <id>  — Execute primary action (auto-executable only; skips confirm-required actions)
```

Tenant isolation: resolved via `bot.ResolveTenantByChat(chatID)`, same as existing commands.

### Cron Job

```go
sched.AddJob("recommendation_scan", "30 10 * * *", recommendationScanFn)
```

---

## Section 6: LLM Prompts

### Daily Scan Prompt

**System:**
```
你是 {mentor_name}，作为 AI 管理顾问分析团队状况。
基于以下数据，生成最多 5 条可执行的管理建议。

规则：
- 每条建议必须有明确的数据支撑（evidence）
- 建议必须可执行（附带 suggested_actions）
- 优先级：critical（需立即处理）> high（今天内）> medium（本周）> low（参考）
- 不要重复已有的 pending 建议
- 用 {culture} 文化的沟通风格
```

**User:** 8 data sections (signals, tasks, KPIs, OKRs, employees, pending recs, memory) + JSON output format spec with full action params schema.

**Token budget:** System prompt ~200 tokens, user data ~16,000 tokens (truncated), output ~4,096 tokens via `ChatLong()`.

### Realtime LLM Prompt (complex triggers only)

**System:**
```
你是 {mentor_name}。根据刚发生的事件，生成一条简短的管理建议。
```

**User:** Event details + employee context + recent signals + memory recall + JSON format spec.

**Token budget:** ~2,000 input tokens, ~1,024 output tokens via `Chat()`.

### Template Generation (simple triggers, no LLM)

Go functions that produce recommendations directly:
- `templateConsecutiveMiss(emp, days)` → people recommendation
- `templateSentimentDrop(emp, trend)` → people recommendation
- `templateTaskOverdue(task, days)` → project recommendation
- `templatePerformanceSpike(emp)` → people recognition recommendation

---

## Sidebar Position

```
── Organize ──
  Dashboard
  AI Recommendations    ← NEW (with pending count badge)
  Team Members
  Organization
  Mentor
  C-Suite Board
```

---

## Estimates

- **New files**: 11
- **Modified files**: 8
- **Total LOC**: ~1200
- **Claude API cost**: 1-3 calls/tenant/day
- **Migration**: 000017

## Reused Components

| Component | Usage |
|-----------|-------|
| `shared/PageHeader.vue` | Page header + actions |
| `shared/EmptyState.vue` | No recommendations state |
| NTabs, NCard, NTag, NButton, NSpace, NEmpty | NaiveUI components |
| `internal/brain/llm.go` | Claude API client (Chat, ChatLong) |
| `internal/scheduler/scheduler.go` | Cron job registration |
| `internal/report/triggers.go` | Event trigger infrastructure |
| `internal/memory/retriever.go` | Memory recall for context |
| `internal/brain/execution_planner.go` | Reuse context-gathering logic from ExecutionPlanner |
