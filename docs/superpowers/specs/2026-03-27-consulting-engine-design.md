# AI Consulting Engine — Design Spec

> **Goal**: Turn AI Management Brain into a McKinsey-style consulting agent that diagnoses problems through conversation, generates structured management plans, executes approved actions, and tracks outcomes with daily progress pushes.

## Core Concept

An **Engagement** is a stateful, multi-turn consulting session spanning minutes to weeks. The boss describes a management problem; the AI walks through a structured consulting methodology: diagnose → analyze → plan → review → execute → track → close.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Interaction | Conversation-driven (Telegram / MCP) | Natural, like talking to a real consultant |
| Depth | Auto-detected (quick/standard/deep) | System assesses complexity from problem + data |
| Execution | Semi-automatic (boss approves each action) | Trust + control balance |
| Tracking | Proactive daily push | Boss stays informed without asking |

## Architecture (Approach C: Thin Wrapper + Fix Dispatcher)

```
┌──────────────────────────────────────┐
│      Consulting Engine (NEW)         │
│  State Machine + Dialogue Orchestr.  │
│  + Action Approval + Progress Track  │
├──────────────────────────────────────┤
│  ContextService │ ExecutionPlanner   │  ← EXISTING: data + analysis
│  Recommender    │ Memory System      │
├──────────────────────────────────────┤
│  Dispatcher (FIX stubs)             │  ← EXISTING: wire to DB
│  create_task ✅ schedule_meeting ✅  │
│  flag_risk ✅   public_recognition  │
├──────────────────────────────────────┤
│  Telegram Bot / MCP / Scheduler     │  ← EXISTING: channels
└──────────────────────────────────────┘
```

**New code**: ~900 LOC across consulting engine, bot commands, MCP tools, scheduler job, migration, sqlc queries.

**Modified code**: ~80 LOC fixing 3 dispatcher stubs + orchestrator intent.

## Engagement Lifecycle (State Machine)

```
INTAKE → DIAGNOSIS → ANALYSIS → PLAN → REVIEW → EXECUTING → TRACKING → CLOSED
```

### Phase Details

**INTAKE** (automatic, 1 message):
- Boss describes problem (NL text)
- AI pulls `ContextService.GetCompanyContext()` + recent memory
- AI classifies tier + category via LLM
- Creates `engagements` record
- Transitions to DIAGNOSIS with first question

**DIAGNOSIS** (2-10 turns based on tier):
- AI generates targeted questions one at a time
- Each question informed by: system data + previous answers + consulting frameworks
- LLM decides per-turn: "ask another question" OR "sufficient info, proceed to analysis"
- Questions + answers stored in `diagnosis_questions` / `diagnosis_answers` JSONB

**ANALYSIS** (automatic, no boss interaction):
- LLM cross-references: diagnosis answers × system data × employee memories × strategy memories
- Applies consulting frameworks based on category:
  - People → Herzberg motivation, Situational Leadership
  - Process → Lean, Theory of Constraints
  - Strategy → Porter's, SWOT
  - Performance → OKR cascade, KPI alignment
- Produces structured root cause analysis
- Stored in `analysis` JSONB

**PLAN** (AI generates, sends to boss):
- Structured management plan with:
  - Diagnosis summary (1-2 sentences)
  - Root causes (ranked)
  - Action items (each with type, owner, priority, deadline, reason)
  - Expected outcomes
  - Risk factors
- Creates `engagement_actions` records (status: pending)
- Sends plan to boss, waits for review

**REVIEW** (boss approves/rejects each action):
- Boss sees numbered action list
- Responds with approvals: "1,2,4 yes / 3 no" or reviews one by one
- Actions marked approved/rejected
- When review complete, transitions to EXECUTING

**EXECUTING** (automatic):
- Dispatcher executes all approved actions:
  - `create_task` → writes to `tasks` table, links back via `linked_task_id`
  - `schedule_meeting` → writes to `meetings` table, links via `linked_meeting_id`
  - `send_message` → sends via channel.Sender
  - `flag_risk` → writes to `execution_signals`
  - `monitor` → creates tracking reminder only
- Results stored in `engagement_actions.result`
- Transitions to TRACKING

**TRACKING** (daily push):
- Scheduler job `engagement_tracker` runs daily at 10:00 AM
- For each active engagement:
  - Check linked task status (done/in_progress/blocked)
  - Check meeting completion
  - Calculate `progress_pct`
  - Push progress message to boss via preferred channel
  - If blocked: suggest plan adjustment
- When all actions complete → suggest closing

**CLOSED** (boss closes or all actions done):
- Generate effectiveness summary
- Store lessons as `strategy_result` memories:
  - Effective actions → importance 0.8
  - Ineffective actions → importance 0.6
- Future engagements pull these memories for smarter plans

## Auto-Tier Classification

LLM receives the problem statement + a data richness summary and returns:

```json
{
  "tier": "quick|standard|deep",
  "category": "people|process|strategy|performance|organization",
  "title": "Short title for this engagement",
  "reasoning": "Why this tier"
}
```

**Heuristics feeding the classification:**
- Single person mentioned + clear data signals → quick
- Team-wide issue + some data → standard
- Organizational/cultural/structural issue + sparse data → deep

## Data Model (Migration 000019)

### Table: `engagements`

| Column | Type | Description |
|--------|------|-------------|
| id | UUID PK | |
| tenant_id | UUID FK | |
| title | TEXT | AI-generated short title |
| problem_statement | TEXT | Original boss input |
| tier | TEXT | quick / standard / deep |
| category | TEXT | people / process / strategy / performance / organization |
| phase | TEXT | intake / diagnosis / analysis / plan / review / executing / tracking / closed |
| diagnosis_questions | JSONB | `[{q: "...", context: "..."}]` |
| diagnosis_answers | JSONB | `[{q: "...", a: "..."}]` |
| diagnosis_data | JSONB | System data snapshot at intake |
| analysis | JSONB | Root cause analysis result |
| plan | JSONB | Full management plan |
| progress_pct | NUMERIC(5,2) | 0-100 |
| next_check_at | TIMESTAMPTZ | Next tracking check |
| mentor_id | TEXT | Active mentor at creation |
| culture_code | TEXT | Active culture at creation |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |
| closed_at | TIMESTAMPTZ | |

### Table: `engagement_actions`

| Column | Type | Description |
|--------|------|-------------|
| id | UUID PK | |
| engagement_id | UUID FK | |
| action_type | TEXT | create_task / schedule_meeting / send_message / flag_risk / monitor / follow_up |
| title | TEXT | Action description |
| description | TEXT | Details |
| params | JSONB | Action parameters |
| owner_name | TEXT | Who is responsible |
| priority | TEXT | critical / high / medium / low |
| due_at | TIMESTAMPTZ | Expected completion |
| status | TEXT | pending / approved / rejected / done / failed |
| approved_at | TIMESTAMPTZ | |
| executed_at | TIMESTAMPTZ | |
| result | JSONB | Execution outcome |
| linked_task_id | UUID FK → tasks | |
| linked_meeting_id | UUID FK → meetings | |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### Indexes

```sql
CREATE INDEX idx_engagements_tenant_phase ON engagements(tenant_id, phase);
CREATE INDEX idx_engagements_next_check ON engagements(next_check_at) WHERE phase IN ('executing', 'tracking');
CREATE INDEX idx_engagement_actions_engagement ON engagement_actions(engagement_id);
CREATE INDEX idx_engagement_actions_status ON engagement_actions(engagement_id, status);
```

## Consulting Engine API

### Go Interface (`internal/brain/consulting.go`)

```go
type ConsultingEngine struct {
    llm            LLMClient
    contextService *ContextService
    dispatcher     *Dispatcher
    queries        *sqlc.Queries
    memStore       *memory.MemoryStore
}

// StartEngagement creates engagement, classifies tier, pulls data, generates first question.
func (ce *ConsultingEngine) StartEngagement(ctx context.Context, tenantID pgtype.UUID, problem string, mentorID, cultureCode string) (*Engagement, string, error)

// AnswerQuestion processes answer, decides: ask more OR transition to analysis+plan.
func (ce *ConsultingEngine) AnswerQuestion(ctx context.Context, engagementID pgtype.UUID, answer string) (nextQuestion string, plan *EngagementPlan, done bool, err error)

// ReviewAction approves or rejects a single action.
func (ce *ConsultingEngine) ReviewAction(ctx context.Context, actionID pgtype.UUID, approved bool) error

// ExecuteApproved dispatches all approved actions and returns results.
func (ce *ConsultingEngine) ExecuteApproved(ctx context.Context, engagementID pgtype.UUID) ([]ActionResult, error)

// CheckProgress evaluates progress on active engagement, returns formatted report.
func (ce *ConsultingEngine) CheckProgress(ctx context.Context, engagementID pgtype.UUID) (string, error)

// CloseEngagement finalizes, stores lessons as memories, returns summary.
func (ce *ConsultingEngine) CloseEngagement(ctx context.Context, engagementID pgtype.UUID) (string, error)
```

## Dispatcher Fixes

Three methods that currently only log need to actually write to DB:

### `createTask` (currently stub)
```go
func (d *Dispatcher) createTask(ctx, tenantID, params) ActionResult {
    // Parse: title, owner_id (resolve by name), priority, due_at, description
    // Call: d.queries.CreateTask(ctx, CreateTaskParams{..., CreatedByAgent: true})
    // Return: ActionResult{Success: true, TaskID: task.ID.String()}
}
```

### `scheduleMeeting` (currently stub)
```go
func (d *Dispatcher) scheduleMeeting(ctx, tenantID, params) ActionResult {
    // Parse: employee_id, agenda/notes, duration (default 30min)
    // Call: d.queries.CreateMeeting(ctx, CreateMeetingParams{...})
    // Return: ActionResult{Success: true, MeetingID: meeting.ID.String()}
}
```

### `flagRisk` (currently stub)
```go
func (d *Dispatcher) flagRisk(ctx, tenantID, params) ActionResult {
    // Parse: risk_description, subject_type, subject_id, severity
    // Call: d.queries.CreateExecutionSignal(ctx, ...)
    // Return: ActionResult{Success: true}
}
```

**ActionResult extended** with optional ID fields:
```go
type ActionResult struct {
    // ... existing fields ...
    TaskID    string `json:"task_id,omitempty"`
    MeetingID string `json:"meeting_id,omitempty"`
}
```

## Bot Integration

### New Commands

```
/consult              — Start a consulting engagement (or resume active one)
/consult status       — View active engagement progress
/consult close        — Close current engagement
/consult history      — List past engagements
```

### Intent Detection

Add `IntentConsult` to Orchestrator. Detected when boss message contains problem-description patterns:
- "最近团队..." / "我发现..." / "怎么解决..."
- "team is struggling" / "I noticed" / "how to handle"
- Sentence length > 20 chars + no command prefix + not in seat/consult mode

When detected: ask boss "Sounds like a management challenge. Want me to run a structured diagnosis? (/consult to start)"

### Active Engagement Routing

```
Redis key: consulting:{tenant_id}:active → engagement_id
Redis key: consulting:{tenant_id}:phase → current phase

When boss sends message:
  1. Check if active engagement exists
  2. If yes AND phase=diagnosis → route to AnswerQuestion
  3. If yes AND phase=review → route to ReviewAction (parse approval)
  4. Otherwise → normal bot flow
```

Boss can `/exit` to pause engagement (resumes with `/consult`).

## MCP Tools (5 new)

### Read Tools (2)

| Tool | Params | Returns |
|------|--------|---------|
| `list_engagements` | status (optional) | All engagements with phase, progress, title |
| `get_engagement_status` | engagement_id | Full engagement detail: phase, actions, progress |

### Write Tools (3)

| Tool | Params | Returns |
|------|--------|---------|
| `start_engagement` | problem_statement | Engagement + first diagnosis question |
| `answer_diagnosis` | engagement_id, answer | Next question or plan |
| `review_engagement_actions` | engagement_id, approvals `[{id, approved}]` | Updated action statuses |

## Scheduler Job

**Name**: `engagement_tracker`
**Schedule**: Daily at 10:00 AM (after check-in reminders)
**Logic**:
1. `SELECT * FROM engagements WHERE phase IN ('executing', 'tracking') AND next_check_at <= now()`
2. For each engagement:
   a. Load all `engagement_actions` with linked_task_id / linked_meeting_id
   b. Check task statuses: `SELECT status FROM tasks WHERE id = ANY($linked_ids)`
   c. Calculate `progress_pct = done_actions / total_approved_actions * 100`
   d. Update `engagements.progress_pct` and `next_check_at = now() + 1 day`
   e. Format progress message and send to boss
   f. If `progress_pct >= 100`: suggest closing

## Memory Integration

### Input (diagnosis phase)
```
memories := memStore.List(ctx, tenantID, "strategy_result", "", "", 10, 0)
// Inject into diagnosis prompt:
// "Past consulting outcomes: [memory summaries]"
```

### Output (close phase)
```
For each action:
  if status == "done":
    memStore.Create(ctx, Memory{Type: "strategy_result", Content: "Action X was effective for problem Y", Importance: 0.8})
  if status == "failed":
    memStore.Create(ctx, Memory{Type: "strategy_result", Content: "Action X did not work for problem Y", Importance: 0.6})
```

## File Impact

### New Files (6)
| File | LOC | Description |
|------|-----|-------------|
| `sql/migrations/000019_engagements.up.sql` | ~60 | 2 tables + indexes |
| `sql/migrations/000019_engagements.down.sql` | ~5 | Drop tables |
| `sql/queries/engagements.sql` | ~80 | CRUD queries |
| `internal/brain/consulting.go` | ~400 | Core engine |
| `internal/brain/consulting_test.go` | ~200 | Unit tests |
| `internal/brain/consulting_prompts.go` | ~100 | LLM prompts for each phase |

### Modified Files (7)
| File | Changes |
|------|---------|
| `internal/brain/dispatcher.go` | Fix 3 stubs (~80 LOC) |
| `internal/brain/orchestrator.go` | Add IntentConsult (~30 LOC) |
| `internal/bot/commands.go` | Add /consult command + routing (~100 LOC) |
| `cmd/brain/main.go` | Wire ConsultingEngine + scheduler job (~20 LOC) |
| `internal/api/router.go` | Add engagement API routes (~15 LOC) |
| `mcp-server/src/server.ts` | Add 5 MCP tools (~120 LOC) |
| `internal/db/sqlc/` | Regenerated from new queries |

**Total**: ~1210 LOC new + ~245 LOC modified

## Non-Goals (v1)

- No web frontend for engagements (conversation-driven only in v1)
- No concurrent engagements per tenant (one active at a time)
- No engagement templates (AI generates fresh each time)
- No multi-tenant engagement sharing
