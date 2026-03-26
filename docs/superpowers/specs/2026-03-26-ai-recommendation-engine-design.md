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
- MCP Streamable HTTP (2 new tools)

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
    target_entity_type TEXT,  -- 'employee', 'project', 'metric', 'goal', NULL
    target_entity_id   UUID, -- FK to the relevant entity
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at     TIMESTAMPTZ,
    executed_at     TIMESTAMPTZ
);

CREATE INDEX idx_recommendations_tenant_status ON recommendations(tenant_id, status);
CREATE INDEX idx_recommendations_tenant_created ON recommendations(tenant_id, created_at DESC);
CREATE INDEX idx_recommendations_expires ON recommendations(expires_at) WHERE status = 'pending';
```

### Suggested Action JSON Schema

```jsonc
[
  {
    "type": "schedule_meeting|send_message|create_task|reassign_task|flag_risk|adjust_target|public_recognition|create_suggestion",
    "params": {
      "employee_id": "uuid",    // varies by type
      "message": "string",
      // ... type-specific params
    },
    "label": "UI button text"
  }
]
```

### Evidence JSON Schema

```jsonc
{
  "signals": ["engagement_drop:0.72", "delivery_risk:0.65"],
  "metrics": ["revenue:declining_2w"],
  "employees": ["Alice:sentiment_drop_3d"],
  "tasks": ["task-123:overdue_5d"]
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

**Process:**
1. Gather all 8 data sources into structured JSON
2. Single Claude API call with mentor-aware system prompt
3. Parse JSON array output (max 5 recommendations)
4. Dedup against pending recommendations (same category + target entity + 72h window)
5. Store in `recommendations` table with `source = 'daily_scan'`
6. Push critical/high priority via Telegram

### Pipeline 2: Real-time Event Triggers

**Trigger points** (extend existing `internal/report/triggers.go`):

| Event | Condition | Template or LLM |
|-------|-----------|-----------------|
| `consecutive_miss` | 3+ days missed | Template |
| `sentiment_drop` | 3 consecutive negative | Template |
| `signal_high_score` | Any signal >= 0.7 | LLM (needs context) |
| `task_overdue_critical` | Critical task 3+ days overdue | Template |
| `blocker_cascade` | 3+ people blocked by same issue | LLM |
| `performance_spike` | Exceptional delivery streak | Template |
| `metric_anomaly` | KPI off-target 2+ periods | LLM |

**Process:**
1. Trigger rule matches (Go code, no LLM)
2. Check dedup (same category + target + 72h)
3. Generate recommendation: template (simple) or lightweight LLM call (complex)
4. Store with `source = 'realtime_trigger'`
5. If priority = critical, push immediately

### Dedup Strategy

- Same `category` + same `target_entity_id` + pending recommendation within 72h → skip
- If new recommendation has higher priority → expire old one, create new

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

### Telegram Format

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

```
get_recommendations      — List pending recommendations
execute_recommendation   — Execute a specific action on a recommendation
```

---

## Section 4: Action Dispatcher

### Location: `internal/brain/dispatcher.go`

### Flow

```
POST /recommendations/:id/execute { action_index: 0 }
  → Load recommendation + verify tenant
  → Extract action by index
  → dispatcher.Execute(ctx, action)
  → Update recommendation status
  → Return result
```

### Dispatch Table

```go
switch action.Type {
case "schedule_meeting":  → meetingQueries.CreateMeeting()
case "send_message":      → sender.Send(channel, message)
case "create_task":       → taskQueries.CreateTask()
case "reassign_task":     → taskQueries.UpdateTask() (with confirmation)
case "flag_risk":         → projectQueries.UpdateProject(risk fields)
case "adjust_target":     → return link to edit page (no execution)
case "public_recognition": → sender.SendToGroup(message)
case "create_suggestion": → orgQueries.CreateSuggestion()
}
```

### Safety

- `reassign_task`: Web UI shows confirmation dialog before executing
- `adjust_target`: Never auto-executes; returns a deeplink to the KPI/OKR edit page
- All other actions: Execute immediately on button click
- Failed execution: recommendation stays `pending`, error message returned to UI

### Result Format

```jsonc
// Success
{ "success": true, "message": "1:1 meeting scheduled for 2026-03-28" }

// Failure
{ "success": false, "error": "Employee has no active channel" }
```

---

## Section 5: Backend Modules & API

### New Files (7)

```
sql/migrations/000016_recommendations.up.sql      — Create table + indexes
sql/migrations/000016_recommendations.down.sql     — Drop table
sql/queries/recommendations.sql                    — sqlc queries (8 queries)
internal/brain/recommender.go                      — DailyScan() + RealtimeEvaluate()
internal/brain/dispatcher.go                       — Action execution dispatcher
internal/api/recommendation_handlers.go            — 6 HTTP handlers
frontend/src/types/recommendation.ts               — TypeScript types
```

### Modified Files (7)

```
internal/api/router.go              — Add recommendations route group
internal/report/triggers.go         — Call recommender.RealtimeEvaluate() on trigger match
internal/api/openclaw.go            — Register 2 new MCP tools
cmd/brain/main.go                   — Register cron job + init recommender + migration 000016
frontend/src/router/index.ts        — Add /recommendations route
frontend/src/layouts/AppLayout.vue  — Sidebar menu item + pending badge
frontend/src/views/DashboardView.vue — Embed RecommendationSummary component
```

### New Frontend Files (5)

```
frontend/src/api/recommendations.ts                        — API client (6 functions)
frontend/src/views/RecommendationsView.vue                  — Full page with tabs
frontend/src/components/recommendations/RecommendationCard.vue  — Single card + actions
frontend/src/components/recommendations/RecommendationSummary.vue — Dashboard embed (top 3)
frontend/src/types/recommendation.ts                        — Types
```

### API Endpoints (6)

```
GET    /api/v1/recommendations              — List (filter: ?status=pending&category=people)
GET    /api/v1/recommendations/summary      — Dashboard: pending count + top 3
POST   /api/v1/recommendations/:id/execute  — Execute one action (body: {action_index})
POST   /api/v1/recommendations/:id/execute-all — Execute all actions
POST   /api/v1/recommendations/:id/dismiss  — Mark dismissed
DELETE /api/v1/recommendations/:id          — Delete (only dismissed/expired)
```

All endpoints require authentication + `RequireRole("boss")`.

### Telegram Commands (2)

```
/recs          — List pending recommendations
/execute <id>  — Execute recommendation (or via inline callback button)
```

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

**User:** 8 data sections (signals, tasks, KPIs, OKRs, employees, pending recs, memory) + JSON output format spec.

### Realtime LLM Prompt (complex triggers only)

**System:**
```
你是 {mentor_name}。根据刚发生的事件，生成一条简短的管理建议。
```

**User:** Event details + employee context + recent signals + memory recall + JSON format spec.

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

- **New files**: 12
- **Modified files**: 7
- **Total LOC**: ~1200
- **Claude API cost**: 1-3 calls/tenant/day
- **Migration**: 000016

## Reused Components

| Component | Usage |
|-----------|-------|
| `shared/PageHeader.vue` | Page header + actions |
| `shared/EmptyState.vue` | No recommendations state |
| NTabs, NCard, NTag, NButton, NSpace, NEmpty | NaiveUI components |
| `internal/brain/llm.go` | Claude API client (ChatLong) |
| `internal/scheduler/scheduler.go` | Cron job registration |
| `internal/report/triggers.go` | Event trigger infrastructure |
| `internal/memory/retriever.go` | Memory recall for context |
