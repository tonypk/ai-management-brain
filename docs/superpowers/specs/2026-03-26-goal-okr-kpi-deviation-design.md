# Goal/OKR Backend + KPI Dashboard + Deviation Tracking — Design Spec

**Target**: Batch 1 of Management 101 feature gap closure

## Problem

The Goals/OKR feature exists in the frontend (`GoalsView.vue`, 8 components, planning store) but stores data in **localStorage only**. This means:

1. Data lost on browser clear / device switch
2. No multi-user visibility (boss can't see team objectives)
3. No deviation tracking (expected vs actual progress)
4. No integration with existing employee/report systems

## Solution

Add backend persistence for Goals/OKR with:
- 3 new database tables: `goals`, `key_results`, `goal_snapshots`
- 8 API endpoints for CRUD + dashboard
- Modify frontend store to use API instead of localStorage
- Daily cron snapshot for deviation tracking
- KPI deviation chart on Goals page

## Database Schema

### Migration: `000012_goals.up.sql`

```sql
CREATE TABLE goals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    owner_id    UUID REFERENCES employees(id),
    title       VARCHAR(500) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'active', 'completed', 'cancelled')),
    cycle       VARCHAR(10) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goals_tenant_cycle ON goals(tenant_id, cycle);
CREATE INDEX idx_goals_owner ON goals(owner_id) WHERE owner_id IS NOT NULL;

CREATE TABLE key_results (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id     UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title       VARCHAR(500) NOT NULL,
    target      NUMERIC(12,2) NOT NULL DEFAULT 0,
    current_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    unit        VARCHAR(20) NOT NULL DEFAULT '%',
    due_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_key_results_goal ON key_results(goal_id);

CREATE TABLE goal_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id         UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    overall_progress NUMERIC(5,2) NOT NULL DEFAULT 0,
    snapshot_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goal_snapshots_goal_date ON goal_snapshots(goal_id, snapshot_date);
CREATE UNIQUE INDEX idx_goal_snapshots_unique ON goal_snapshots(goal_id, snapshot_date);
```

**Design decisions:**
- `goals.status` uses same values as frontend: `draft`, `active`, `completed`, `cancelled`
- `goals.cycle` is VARCHAR(10) for `"2026-Q1"` format — matches frontend `GoalCycle` type
- `goals.owner_id` is nullable — goal can be unassigned (team-level OKR)
- `key_results.target`/`current_value` use NUMERIC(12,2) not INT — supports percentages, currency, fractional metrics
- `key_results.current_value` instead of `current` to avoid PostgreSQL reserved word
- `goal_snapshots` stores daily progress per goal — one row per goal per day, unique constraint prevents duplicates
- CASCADE delete: deleting a goal removes its key_results and snapshots

### Down migration: `000012_goals.down.sql`

```sql
DROP TABLE IF EXISTS goal_snapshots;
DROP TABLE IF EXISTS key_results;
DROP TABLE IF EXISTS goals;
```

## SQL Queries

### File: `sql/queries/goals.sql`

```sql
-- name: CreateGoal :one
INSERT INTO goals (tenant_id, owner_id, title, description, status, cycle)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListGoalsByCycle :many
SELECT g.*,
       COALESCE(
         (SELECT json_agg(kr ORDER BY kr.created_at)
          FROM key_results kr WHERE kr.goal_id = g.id),
         '[]'
       ) AS key_results_json
FROM goals g
WHERE g.tenant_id = $1 AND g.cycle = $2
ORDER BY g.created_at;

-- name: GetGoal :one
SELECT * FROM goals WHERE id = $1 AND tenant_id = $2;

-- name: UpdateGoal :one
UPDATE goals
SET title = $3, description = $4, status = $5, cycle = $6, owner_id = $7, updated_at = now()
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteGoal :exec
DELETE FROM goals WHERE id = $1 AND tenant_id = $2;

-- name: CreateKeyResult :one
INSERT INTO key_results (goal_id, title, target, current_value, unit, due_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateKeyResult :exec
UPDATE key_results
SET title = $2, target = $3, current_value = $4, unit = $5, due_date = $6, updated_at = now()
WHERE id = $1;

-- name: DeleteKeyResult :exec
DELETE FROM key_results WHERE id = $1;

-- name: CreateGoalSnapshot :exec
INSERT INTO goal_snapshots (goal_id, overall_progress, snapshot_date)
VALUES ($1, $2, $3)
ON CONFLICT (goal_id, snapshot_date) DO UPDATE SET overall_progress = EXCLUDED.overall_progress;

-- name: ListGoalSnapshots :many
SELECT * FROM goal_snapshots
WHERE goal_id = $1
ORDER BY snapshot_date;

-- name: ListActiveGoalsByTenant :many
SELECT g.id, g.title
FROM goals g
WHERE g.tenant_id = $1 AND g.status = 'active';

-- name: GetKeyResultsByGoal :many
SELECT * FROM key_results WHERE goal_id = $1 ORDER BY created_at;

-- name: ListTenantsWithActiveGoals :many
SELECT DISTINCT g.tenant_id
FROM goals g
WHERE g.status = 'active';
```

**Key result tenant isolation**: For KR update/delete, the handler first fetches the parent goal by `goal_id` (from URL `:id`), verifies `tenant_id` matches JWT context, then proceeds with the KR mutation. This avoids adding `tenant_id` to the `key_results` table while maintaining security.

## API Endpoints

All under `/api/v1/goals`, protected by JWT + `RequireRole("boss")`.

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/goals?cycle=2026-Q1` | `handleListGoals` | List goals with KRs for a cycle |
| POST | `/goals` | `handleCreateGoal` | Create a new goal |
| PUT | `/goals/:id` | `handleUpdateGoal` | Update goal fields |
| DELETE | `/goals/:id` | `handleDeleteGoal` | Delete goal (cascades KRs + snapshots) |
| POST | `/goals/:id/key-results` | `handleCreateKeyResult` | Add KR to goal |
| PUT | `/goals/:id/key-results/:kr_id` | `handleUpdateKeyResult` | Update KR (including `current` for progress) |
| DELETE | `/goals/:id/key-results/:kr_id` | `handleDeleteKeyResult` | Delete KR |
| GET | `/goals/:id/snapshots` | `handleListSnapshots` | Get deviation data for chart |

### Request/Response Shapes

**POST /goals**
```json
// Request
{ "title": "Increase MRR", "description": "...", "cycle": "2026-Q1", "owner_id": "uuid-or-null", "status": "draft" }
// Response
{ "data": { "id": "...", "tenant_id": "...", "title": "...", ... } }
```

**GET /goals?cycle=2026-Q1**
```json
{
  "data": [
    {
      "id": "...", "title": "...", "status": "active", "cycle": "2026-Q1",
      "owner_id": "...",
      "key_results": [
        { "id": "...", "title": "...", "target": 100, "current": 65, "unit": "%", "due_date": "2026-03-31" }
      ]
    }
  ]
}
```

**GET /goals/:id/snapshots**
```json
{
  "data": [
    { "snapshot_date": "2026-03-01", "overall_progress": 12.5 },
    { "snapshot_date": "2026-03-02", "overall_progress": 15.0 }
  ]
}
```

### Handler Pattern

Follows existing codebase pattern (`org_handlers.go`):
```go
func handleCreateGoal(queries *sqlc.Queries) gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID, err := parseUUID(TenantFromContext(c))
        // ... validate, create, return
    }
}
```

### Tenant Scoping

- `CreateGoal`: receives `tenant_id` from JWT context
- `ListGoalsByCycle`: filters by `tenant_id` + `cycle` query param
- `GetGoal`/`UpdateGoal`/`DeleteGoal`: SQL WHERE includes `tenant_id` — defense-in-depth at DB level
- `CreateKeyResult`/`UpdateKeyResult`/`DeleteKeyResult`: handler fetches parent goal first, verifies `tenant_id` matches JWT before proceeding
- `ListSnapshots`: handler fetches parent goal first, verifies `tenant_id` matches JWT before returning snapshot data
- All mutations verify tenant ownership to prevent cross-tenant access

## Go Handler File

New file: `internal/api/goal_handlers.go`

Request structs:
```go
type createGoalRequest struct {
    Title       string  `json:"title" binding:"required"`
    Description string  `json:"description"`
    Cycle       string  `json:"cycle" binding:"required"`
    OwnerID     *string `json:"owner_id"`
    Status      string  `json:"status"` // defaults to "draft" in handler if empty
}

type updateGoalRequest struct {
    Title       string  `json:"title" binding:"required"`
    Description string  `json:"description"`
    Cycle       string  `json:"cycle" binding:"required"`
    OwnerID     *string `json:"owner_id"`
    Status      string  `json:"status" binding:"required,oneof=draft active completed cancelled"`
}

type createKeyResultRequest struct {
    Title        string  `json:"title" binding:"required"`
    Target       float64 `json:"target" binding:"required,gt=0"`
    CurrentValue float64 `json:"current_value"`
    Unit         string  `json:"unit"`
    DueDate      *string `json:"due_date"`
}

type updateKeyResultRequest struct {
    Title        string  `json:"title" binding:"required"`
    Target       float64 `json:"target" binding:"required,gt=0"`
    CurrentValue float64 `json:"current_value"`
    Unit         string  `json:"unit"`
    DueDate      *string `json:"due_date"`
}
```

**Status handling**: `createGoalRequest.Status` defaults to `"draft"` in the handler when empty. `updateGoalRequest.Status` requires one of the 4 valid values.

**Target validation**: `gt=0` ensures target is positive — a zero target would cause division-by-zero in progress calculation.

## Router Registration

In `router.go`, add goals group after org group:

```go
// Goals/OKR
goals := protected.Group("/goals")
goals.Use(RequireRole("boss"))
{
    goals.GET("", handleListGoals(cfg.Queries))
    goals.POST("", handleCreateGoal(cfg.Queries))
    goals.PUT("/:id", handleUpdateGoal(cfg.Queries))
    goals.DELETE("/:id", handleDeleteGoal(cfg.Queries))
    goals.POST("/:id/key-results", handleCreateKeyResult(cfg.Queries))
    goals.PUT("/:id/key-results/:kr_id", handleUpdateKeyResult(cfg.Queries))
    goals.DELETE("/:id/key-results/:kr_id", handleDeleteKeyResult(cfg.Queries))
    goals.GET("/:id/snapshots", handleListSnapshots(cfg.Queries))
}
```

## Cron Job: Daily Snapshots

Register in `cmd/brain/main.go` after existing cron jobs:

```go
sched.AddJob("goal_snapshots", "0 23 * * *", func(ctx context.Context) error {
    // 1. ListTenantsWithActiveGoals() → get distinct tenant_ids
    // 2. For each tenant_id:
    //    a. ListActiveGoalsByTenant(tenant_id) → get goal IDs
    //    b. For each goal: GetKeyResultsByGoal(goal_id) → calculate progress
    //    c. CreateGoalSnapshot(goal_id, progress, today)
})
```

Runs at 23:00 daily (scheduler timezone). Uses `ListTenantsWithActiveGoals` query to enumerate tenants. Progress formula per goal:
- If goal has 0 KRs: progress = 0
- Otherwise: average of `min(current_value/target * 100, 100)` across all KRs

## Frontend Changes

### New file: `frontend/src/api/goals.ts`

```typescript
import { get, post, put, del } from './client'

interface GoalResponse { /* matches backend response */ }
interface KeyResultResponse { /* matches backend response */ }
interface SnapshotResponse { /* matches backend response */ }

export async function listGoals(cycle: string): Promise<GoalResponse[]> { ... }
export async function createGoal(data: CreateGoalInput): Promise<GoalResponse> { ... }
export async function updateGoal(id: string, data: UpdateGoalInput): Promise<void> { ... }
export async function deleteGoal(id: string): Promise<void> { ... }
export async function createKeyResult(goalId: string, data: CreateKRInput): Promise<KeyResultResponse> { ... }
export async function updateKeyResult(goalId: string, krId: string, data: UpdateKRInput): Promise<void> { ... }
export async function deleteKeyResult(goalId: string, krId: string): Promise<void> { ... }
export async function listSnapshots(goalId: string): Promise<SnapshotResponse[]> { ... }
```

### Modify: `frontend/src/types/planning.ts`

Update `KeyResult` to rename `current` → `current_value` and make `due_date` nullable. Add `owner_id` to `Objective`:
```typescript
export interface KeyResult {
  id: string
  title: string
  target: number
  current_value: number  // renamed from "current"
  unit: string
  due_date: string | null  // nullable DATE from backend
}

export interface Objective {
  id: string
  title: string
  description: string
  status: GoalStatus
  cycle: GoalCycle
  owner_id: string | null  // NEW
  key_results: KeyResult[]
  created_at: string
  updated_at: string
}
```

### Modify: `frontend/src/stores/planning.ts`

Replace localStorage-based goals section with API calls:
- `loadGoals(cycle)` → `GET /goals?cycle=X`
- `addObjective(...)` → `POST /goals` + refresh
- `updateObjective(...)` → `PUT /goals/:id` + refresh
- `deleteObjective(...)` → `DELETE /goals/:id` + refresh
- `addKeyResult(...)` → `POST /goals/:id/key-results` + refresh
- `updateKeyResult(...)` → `PUT /goals/:id/key-results/:kr_id` + refresh
- `deleteKeyResult(...)` → `DELETE /goals/:id/key-results/:kr_id` + refresh
- Keep `cycleStats()` as computed from loaded data
- Add `loading` ref for loading state
- Board records stay in localStorage (separate feature)

### Modify: `frontend/src/views/GoalsView.vue`

- Call `store.loadGoals(cycle)` on mount and cycle change
- Add loading spinner during API calls
- Add owner selector (dropdown of employees) in ObjectiveFormModal
- Add deviation chart section using snapshot data

### New component: `GoalDeviationChart.vue`

Simple line chart showing:
- X-axis: dates (from snapshots)
- Y-axis: progress %
- One line per selected goal
- Expected progress line (linear from 0% to 100% across cycle dates)
- Uses NaiveUI + basic SVG or existing chart lib if available

## Files to Create/Modify

### New files (6)
| File | Purpose | ~LOC |
|------|---------|------|
| `sql/migrations/000012_goals.up.sql` | Schema: goals, key_results, goal_snapshots | 35 |
| `sql/migrations/000012_goals.down.sql` | Rollback | 5 |
| `sql/queries/goals.sql` | sqlc queries | 65 |
| `internal/api/goal_handlers.go` | 8 HTTP handlers | 250 |
| `frontend/src/api/goals.ts` | API client module | 55 |
| `frontend/src/components/goals/GoalDeviationChart.vue` | Deviation chart | 80 |

### Modified files (4)
| File | Change | ~Lines |
|------|--------|--------|
| `internal/api/router.go` | Add goals route group | +12 |
| `cmd/brain/main.go` | Add snapshot cron job | +30 |
| `frontend/src/types/planning.ts` | Add `owner_id` to Objective | +1 |
| `frontend/src/stores/planning.ts` | Replace localStorage with API calls | ~80 changed |

**Total: 6 new files + 4 modified, ~610 LOC**

## YAGNI Exclusions

Not building:
- Goal alignment / cascading (parent-child goals)
- Budget tracking
- Burndown charts
- Milestone tracking
- Goal templates
- Goal comments/activity feed
- Goal sharing permissions (all goals visible to tenant boss)
- Weighted KR progress (all KRs equal weight)

## Validation

1. `go build ./...` — no compilation errors
2. `sqlc generate` — generates clean Go code
3. Migration applies cleanly on existing DB
4. `POST /goals` creates goal, visible in `GET /goals?cycle=X`
5. `POST /goals/:id/key-results` adds KR, visible in goal listing
6. `PUT /goals/:id/key-results/:kr_id` with `current` updates progress
7. `DELETE /goals/:id` cascades to KRs and snapshots
8. Cron job creates snapshots for active goals
9. `GET /goals/:id/snapshots` returns deviation data
10. Frontend loads goals from API (not localStorage)
11. Deviation chart renders with snapshot data
12. Frontend deploy: `npm run build` succeeds with no TS errors
