# Goal/OKR Backend + KPI Deviation Tracking — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace localStorage-based Goals/OKR frontend with backend-persisted API, add daily deviation snapshots, and integrate a KPI deviation chart.

**Architecture:** PostgreSQL tables (goals, key_results, goal_snapshots) → sqlc-generated Go code → Gin HTTP handlers → Vue3+NaiveUI frontend. Cron job snapshots daily progress for deviation tracking.

**Tech Stack:** Go 1.25 (Gin, sqlc, pgx/v5), PostgreSQL 16, Vue3+TS+NaiveUI, Pinia

**Spec:** `docs/superpowers/specs/2026-03-26-goal-okr-kpi-deviation-design.md`

---

### Task 1: Database Migration

**Files:**
- Create: `sql/migrations/000012_goals.up.sql`
- Create: `sql/migrations/000012_goals.down.sql`

- [ ] **Step 1: Create up migration**

```sql
-- sql/migrations/000012_goals.up.sql
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
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id       UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title         VARCHAR(500) NOT NULL,
    target        NUMERIC(12,2) NOT NULL DEFAULT 0,
    current_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    unit          VARCHAR(20) NOT NULL DEFAULT '%',
    due_date      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_key_results_goal ON key_results(goal_id);

CREATE TABLE goal_snapshots (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id          UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    overall_progress NUMERIC(5,2) NOT NULL DEFAULT 0,
    snapshot_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goal_snapshots_goal_date ON goal_snapshots(goal_id, snapshot_date);
CREATE UNIQUE INDEX idx_goal_snapshots_unique ON goal_snapshots(goal_id, snapshot_date);
```

- [ ] **Step 2: Create down migration**

```sql
-- sql/migrations/000012_goals.down.sql
DROP TABLE IF EXISTS goal_snapshots;
DROP TABLE IF EXISTS key_results;
DROP TABLE IF EXISTS goals;
```

- [ ] **Step 3: Commit**

```bash
git add sql/migrations/000012_goals.up.sql sql/migrations/000012_goals.down.sql
git commit -m "feat: add goals, key_results, goal_snapshots migration (000012)"
```

---

### Task 2: SQL Queries + sqlc Generate

**Files:**
- Create: `sql/queries/goals.sql`
- Regenerate: `internal/db/sqlc/` (auto-generated)

- [ ] **Step 1: Create goals query file**

```sql
-- sql/queries/goals.sql

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

- [ ] **Step 2: Run sqlc generate**

Run: `cd /Users/anna/Documents/ai-management-brain && ~/go/bin/sqlc generate`
Expected: No errors, new files generated in `internal/db/sqlc/`

- [ ] **Step 3: Verify Go build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add sql/queries/goals.sql internal/db/sqlc/
git commit -m "feat: add sqlc queries for goals, key_results, snapshots"
```

---

### Task 3: Go HTTP Handlers

**Files:**
- Create: `internal/api/goal_handlers.go`
- Modify: `internal/api/router.go` (add route group)

- [ ] **Step 1: Create goal_handlers.go**

Create `/Users/anna/Documents/ai-management-brain/internal/api/goal_handlers.go` with:

```go
package api

import (
	"log/slog"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tonypk/ai-management-brain/internal/db/sqlc"
)

// --- Request types ---

type createGoalRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	Cycle       string  `json:"cycle" binding:"required"`
	OwnerID     *string `json:"owner_id"`
	Status      string  `json:"status"`
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

// --- Handlers ---

func handleListGoals(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		cycle := c.Query("cycle")
		if cycle == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cycle query parameter is required"})
			return
		}

		rows, err := queries.ListGoalsByCycle(c.Request.Context(), sqlc.ListGoalsByCycleParams{
			TenantID: tenantID,
			Cycle:    cycle,
		})
		if err != nil {
			slog.Error("list goals", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": rows})
	}
}

func handleCreateGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		var req createGoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status := req.Status
		if status == "" {
			status = "draft"
		}

		var ownerID pgtype.UUID
		if req.OwnerID != nil {
			parsed, err := parseUUID(*req.OwnerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_id"})
				return
			}
			ownerID = pgtype.UUID{Bytes: parsed.Bytes, Valid: true}
		}

		goal, err := queries.CreateGoal(c.Request.Context(), sqlc.CreateGoalParams{
			TenantID:    tenantID,
			OwnerID:     ownerID,
			Title:       req.Title,
			Description: req.Description,
			Status:      status,
			Cycle:       req.Cycle,
		})
		if err != nil {
			slog.Error("create goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": goal})
	}
}

func handleUpdateGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		goalID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
			return
		}

		var req updateGoalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ownerID pgtype.UUID
		if req.OwnerID != nil {
			parsed, err := parseUUID(*req.OwnerID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid owner_id"})
				return
			}
			ownerID = pgtype.UUID{Bytes: parsed.Bytes, Valid: true}
		}

		goal, err := queries.UpdateGoal(c.Request.Context(), sqlc.UpdateGoalParams{
			ID:          goalID,
			TenantID:    tenantID,
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Cycle:       req.Cycle,
			OwnerID:     ownerID,
		})
		if err != nil {
			slog.Error("update goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": goal})
	}
}

func handleDeleteGoal(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := parseUUID(TenantFromContext(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
			return
		}

		goalID, err := parseUUID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
			return
		}

		if err := queries.DeleteGoal(c.Request.Context(), sqlc.DeleteGoalParams{
			ID:       goalID,
			TenantID: tenantID,
		}); err != nil {
			slog.Error("delete goal", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "deleted"})
	}
}

// verifyGoalTenant checks that a goal belongs to the requesting tenant.
// Used by KR and snapshot handlers for tenant isolation.
func verifyGoalTenant(c *gin.Context, queries *sqlc.Queries) (pgtype.UUID, pgtype.UUID, bool) {
	tenantID, err := parseUUID(TenantFromContext(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	goalID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal id"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	// Verify goal belongs to tenant
	_, err = queries.GetGoal(c.Request.Context(), sqlc.GetGoalParams{
		ID:       goalID,
		TenantID: tenantID,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return pgtype.UUID{}, pgtype.UUID{}, false
	}

	return goalID, tenantID, true
}

func handleCreateKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		var req createKeyResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		unit := req.Unit
		if unit == "" {
			unit = "%"
		}

		var dueDate pgtype.Date
		if req.DueDate != nil {
			parsed, err := parseDateString(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format (use YYYY-MM-DD)"})
				return
			}
			dueDate = parsed
		}

		kr, err := queries.CreateKeyResult(c.Request.Context(), sqlc.CreateKeyResultParams{
			GoalID:       goalID,
			Title:        req.Title,
			Target:       numericFromFloat(req.Target),
			CurrentValue: numericFromFloat(req.CurrentValue),
			Unit:         unit,
			DueDate:      dueDate,
		})
		if err != nil {
			slog.Error("create key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": kr})
	}
}

func handleUpdateKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		krID, err := parseUUID(c.Param("kr_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key result id"})
			return
		}

		var req updateKeyResultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		unit := req.Unit
		if unit == "" {
			unit = "%"
		}

		var dueDate pgtype.Date
		if req.DueDate != nil {
			parsed, err := parseDateString(*req.DueDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_date format (use YYYY-MM-DD)"})
				return
			}
			dueDate = parsed
		}

		if err := queries.UpdateKeyResult(c.Request.Context(), sqlc.UpdateKeyResultParams{
			ID:           krID,
			Title:        req.Title,
			Target:       numericFromFloat(req.Target),
			CurrentValue: numericFromFloat(req.CurrentValue),
			Unit:         unit,
			DueDate:      dueDate,
		}); err != nil {
			slog.Error("update key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "updated"})
	}
}

func handleDeleteKeyResult(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		krID, err := parseUUID(c.Param("kr_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key result id"})
			return
		}

		if err := queries.DeleteKeyResult(c.Request.Context(), krID); err != nil {
			slog.Error("delete key result", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": "deleted"})
	}
}

func handleListSnapshots(queries *sqlc.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, _, ok := verifyGoalTenant(c, queries)
		if !ok {
			return
		}

		snapshots, err := queries.ListGoalSnapshots(c.Request.Context(), goalID)
		if err != nil {
			slog.Error("list snapshots", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": snapshots})
	}
}

// --- Helpers ---

// CalculateGoalProgress computes overall progress for a goal from its key results.
// Returns 0 if no key results exist.
// Exported for use by the cron job in main.go.
func CalculateGoalProgress(krs []sqlc.KeyResult) float64 {
	if len(krs) == 0 {
		return 0
	}
	var sum float64
	for _, kr := range krs {
		target := numericToFloat(kr.Target)
		current := numericToFloat(kr.CurrentValue)
		if target > 0 {
			sum += math.Min(current/target*100, 100)
		}
	}
	return math.Round(sum/float64(len(krs))*100) / 100
}
```

Note: This handler uses helper functions `parseUUID`, `formatUUID`, `TenantFromContext` that already exist in the codebase. It also needs `numericFromFloat`, `numericToFloat`, and `parseDateString` helpers — add these to the handler file or a shared helpers file if they don't exist yet. Check the existing codebase for pgtype.Numeric conversion patterns.

- [ ] **Step 2: Add routes to router.go**

In `internal/api/router.go`, add after the org group (after line 132):

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

- [ ] **Step 3: Verify Go build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add internal/api/goal_handlers.go internal/api/router.go
git commit -m "feat: add goal/OKR HTTP handlers and routes"
```

---

### Task 4: Cron Job for Daily Snapshots

**Files:**
- Modify: `cmd/brain/main.go`

- [ ] **Step 1: Add goal snapshot cron job**

In `cmd/brain/main.go`, find where other cron jobs are added (after `sched.AddJob("weekly_actions", ...)`) and add:

```go
if err := sched.AddJob("goal_snapshots", "0 23 * * *", func(ctx context.Context) error {
    slog.Info("goal snapshots job: calculating daily progress")

    tenantIDs, err := botDB.ListTenantsWithActiveGoals(ctx)
    if err != nil {
        return fmt.Errorf("list tenants with active goals: %w", err)
    }

    today := pgtype.Date{Time: time.Now().Truncate(24 * time.Hour), Valid: true}
    var snapshotCount int

    for _, tenantID := range tenantIDs {
        goals, err := botDB.ListActiveGoalsByTenant(ctx, tenantID)
        if err != nil {
            slog.Error("list active goals", "tenant_id", tenantID, "error", err)
            continue
        }

        for _, goal := range goals {
            krs, err := botDB.GetKeyResultsByGoal(ctx, goal.ID)
            if err != nil {
                slog.Error("get key results", "goal_id", goal.ID, "error", err)
                continue
            }

            progress := api.CalculateGoalProgress(krs)

            if err := botDB.CreateGoalSnapshot(ctx, sqlc.CreateGoalSnapshotParams{
                GoalID:          goal.ID,
                OverallProgress: numericFromFloat(progress),
                SnapshotDate:    today,
            }); err != nil {
                slog.Error("create goal snapshot", "goal_id", goal.ID, "error", err)
                continue
            }
            snapshotCount++
        }
    }

    slog.Info("goal snapshots job: done", "snapshots_created", snapshotCount)
    return nil
}); err != nil {
    slog.Error("failed to register goal_snapshots job", "error", err)
    os.Exit(1)
}
```

Note: The `numericFromFloat` helper may need to be in a shared package or duplicated. Check the existing codebase for the pattern used by other cron jobs that work with NUMERIC types. If no shared helper exists, add the conversion inline.

- [ ] **Step 2: Verify Go build**

Run: `cd /Users/anna/Documents/ai-management-brain && go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add cmd/brain/main.go
git commit -m "feat: add daily goal_snapshots cron job"
```

---

### Task 5: Frontend API + Type Updates

**Files:**
- Create: `frontend/src/api/goals.ts`
- Modify: `frontend/src/types/planning.ts`
- Modify: `frontend/src/api/index.ts` (if exists, to re-export)

- [ ] **Step 1: Update types**

In `frontend/src/types/planning.ts`, update `KeyResult` and `Objective`:

Change `KeyResult.current` to `current_value` and `due_date` to nullable:
```typescript
export interface KeyResult {
  id: string
  title: string
  target: number
  current_value: number
  unit: string
  due_date: string | null
}
```

Add `owner_id` to `Objective`:
```typescript
export interface Objective {
  id: string
  title: string
  description: string
  status: GoalStatus
  cycle: GoalCycle
  owner_id: string | null
  key_results: KeyResult[]
  created_at: string
  updated_at: string
}
```

Add snapshot type:
```typescript
export interface GoalSnapshot {
  id: string
  goal_id: string
  overall_progress: number
  snapshot_date: string
  created_at: string
}
```

- [ ] **Step 2: Create API module**

Create `frontend/src/api/goals.ts`:

```typescript
import { get, post, put, del } from './client'
import type { Objective, KeyResult, GoalSnapshot } from '@/types'

export async function listGoals(cycle: string): Promise<Objective[]> {
  const res = await get<{ data: Objective[] }>(`/goals?cycle=${encodeURIComponent(cycle)}`)
  return res.data
}

export async function createGoal(data: {
  title: string
  description: string
  cycle: string
  owner_id?: string | null
  status?: string
}): Promise<Objective> {
  const res = await post<{ data: Objective }>('/goals', data)
  return res.data
}

export async function updateGoal(id: string, data: {
  title: string
  description: string
  cycle: string
  owner_id?: string | null
  status: string
}): Promise<Objective> {
  const res = await put<{ data: Objective }>(`/goals/${id}`, data)
  return res.data
}

export async function deleteGoal(id: string): Promise<void> {
  await del<{ data: string }>(`/goals/${id}`)
}

export async function createKeyResult(goalId: string, data: {
  title: string
  target: number
  current_value?: number
  unit?: string
  due_date?: string | null
}): Promise<KeyResult> {
  const res = await post<{ data: KeyResult }>(`/goals/${goalId}/key-results`, data)
  return res.data
}

export async function updateKeyResult(goalId: string, krId: string, data: {
  title: string
  target: number
  current_value: number
  unit?: string
  due_date?: string | null
}): Promise<void> {
  await put<{ data: string }>(`/goals/${goalId}/key-results/${krId}`, data)
}

export async function deleteKeyResult(goalId: string, krId: string): Promise<void> {
  await del<{ data: string }>(`/goals/${goalId}/key-results/${krId}`)
}

export async function listSnapshots(goalId: string): Promise<GoalSnapshot[]> {
  const res = await get<{ data: GoalSnapshot[] }>(`/goals/${goalId}/snapshots`)
  return res.data
}
```

- [ ] **Step 3: Export from api/index.ts if it exists**

Check if `frontend/src/api/index.ts` exists and add goals export.

- [ ] **Step 4: Verify TypeScript build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npx tsc --noEmit`
Expected: No errors (or only pre-existing ones)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/planning.ts frontend/src/api/goals.ts
git commit -m "feat: add goals API client and update types for backend integration"
```

---

### Task 6: Frontend Store Refactor (localStorage → API)

**Files:**
- Modify: `frontend/src/stores/planning.ts`

- [ ] **Step 1: Rewrite goals section of planning store**

Replace the localStorage-based goals section with API-backed implementation. Keep board records in localStorage (separate feature). The store should:

1. Add `loading` ref
2. Add `currentCycle` ref
3. Replace all localStorage operations with API calls
4. `loadGoals(cycle)` fetches from API and sets `objectives` ref
5. All mutation functions call API then re-fetch
6. Keep `cycleStats()` as computed from loaded data
7. Keep `objectivesByCycle()` as filter on loaded data

Key changes:
- `objectives` changes from computed off localStorage to a plain `ref<Objective[]>([])`
- All add/update/delete functions become `async` and call the API
- New `loadGoals(cycle: string)` function
- Remove `GOALS_KEY` localStorage constant and `defaultGoalsStorage()`
- Keep `uid()` and `now()` helpers for board records

```typescript
// Goals section replacement (inside the store):
const objectives = ref<Objective[]>([])
const goalsLoading = ref(false)
const currentCycle = ref('')

async function loadGoals(cycle: string) {
  goalsLoading.value = true
  try {
    currentCycle.value = cycle
    objectives.value = await goalsApi.listGoals(cycle)
  } finally {
    goalsLoading.value = false
  }
}

function objectivesByCycle(cycle: GoalCycle): Objective[] {
  return objectives.value.filter((o) => o.cycle === cycle)
}

async function addObjective(title: string, description: string, cycle: GoalCycle, status: GoalStatus = 'draft', ownerId?: string | null): Promise<Objective> {
  const goal = await goalsApi.createGoal({ title, description, cycle, owner_id: ownerId, status })
  await loadGoals(currentCycle.value)
  return goal
}

async function updateObjective(id: string, patch: Partial<Pick<Objective, 'title' | 'description' | 'status' | 'cycle' | 'owner_id'>>): Promise<void> {
  const existing = objectives.value.find(o => o.id === id)
  if (!existing) return
  await goalsApi.updateGoal(id, {
    title: patch.title ?? existing.title,
    description: patch.description ?? existing.description,
    cycle: patch.cycle ?? existing.cycle,
    status: patch.status ?? existing.status,
    owner_id: patch.owner_id !== undefined ? patch.owner_id : existing.owner_id,
  })
  await loadGoals(currentCycle.value)
}

async function deleteObjective(id: string): Promise<void> {
  await goalsApi.deleteGoal(id)
  await loadGoals(currentCycle.value)
}

async function addKeyResult(objectiveId: string, title: string, target: number, unit: string, dueDate: string): Promise<void> {
  await goalsApi.createKeyResult(objectiveId, { title, target, unit, due_date: dueDate || null })
  await loadGoals(currentCycle.value)
}

async function updateKeyResult(objectiveId: string, krId: string, patch: Partial<Pick<KeyResult, 'title' | 'target' | 'current_value' | 'unit' | 'due_date'>>): Promise<void> {
  const obj = objectives.value.find(o => o.id === objectiveId)
  const kr = obj?.key_results.find(k => k.id === krId)
  if (!kr) return
  await goalsApi.updateKeyResult(objectiveId, krId, {
    title: patch.title ?? kr.title,
    target: patch.target ?? kr.target,
    current_value: patch.current_value ?? kr.current_value,
    unit: patch.unit ?? kr.unit,
    due_date: patch.due_date !== undefined ? patch.due_date : kr.due_date,
  })
  await loadGoals(currentCycle.value)
}

async function deleteKeyResult(objectiveId: string, krId: string): Promise<void> {
  await goalsApi.deleteKeyResult(objectiveId, krId)
  await loadGoals(currentCycle.value)
}

function cycleStats(cycle: GoalCycle) {
  const objs = objectivesByCycle(cycle)
  const total = objs.length
  const active = objs.filter((o) => o.status === 'active').length
  const completed = objs.filter((o) => o.status === 'completed').length

  let progress = 0
  if (total > 0) {
    const objProgresses = objs.map((o) => {
      if (o.key_results.length === 0) return 0
      const krSum = o.key_results.reduce((acc, kr) => {
        const p = kr.target > 0 ? Math.min((kr.current_value / kr.target) * 100, 100) : 0
        return acc + p
      }, 0)
      return krSum / o.key_results.length
    })
    progress = Math.round(objProgresses.reduce((a, b) => a + b, 0) / total)
  }

  return { total, active, completed, progress }
}
```

- [ ] **Step 2: Update all component references from `current` to `current_value`**

Search all goal components for `kr.current` or `.current` references and update to `current_value`. Files to check:
- `frontend/src/components/goals/KeyResultList.vue`
- `frontend/src/components/goals/KeyResultFormModal.vue`
- `frontend/src/components/goals/GoalProgressChart.vue`
- `frontend/src/components/goals/GoalOverviewStats.vue`
- `frontend/src/components/goals/ObjectiveCard.vue`

- [ ] **Step 3: Update GoalsView.vue**

- Call `store.loadGoals(cycle)` on mount and when cycle changes
- Add loading state (NaiveUI NSpin or loading overlay)
- Handle async operations in event handlers

- [ ] **Step 4: Verify TypeScript build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 5: Verify Vite build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/planning.ts frontend/src/views/GoalsView.vue frontend/src/components/goals/
git commit -m "feat: migrate goals store from localStorage to backend API"
```

---

### Task 7: Deviation Chart Component

**Files:**
- Create: `frontend/src/components/goals/GoalDeviationChart.vue`
- Modify: `frontend/src/views/GoalsView.vue` (integrate chart)

- [ ] **Step 1: Create GoalDeviationChart component**

Create `frontend/src/components/goals/GoalDeviationChart.vue`:

A simple line chart using SVG (no external chart library needed):
- Props: `goalId: string`, `cycle: string`
- Fetches snapshots via `listSnapshots(goalId)` on mount
- Draws SVG polyline for actual progress
- Draws dashed line for expected progress (linear from 0 to 100 across quarter)
- Color-coded: green if ahead, red if behind
- NaiveUI NCard wrapper with title "Progress Trend"
- Empty state if no snapshots yet

Expected progress calculation:
- Quarter start = first day of quarter (Q1=Jan 1, Q2=Apr 1, Q3=Jul 1, Q4=Oct 1)
- Quarter end = last day of quarter
- Expected progress at date X = (days elapsed / total days) * 100

- [ ] **Step 2: Integrate chart into GoalsView**

Add the deviation chart to `GoalsView.vue`:
- Show chart below the objectives grid
- Allow selecting which goal to show deviation for
- Only show for goals with `active` status

- [ ] **Step 3: Verify build**

Run: `cd /Users/anna/Documents/ai-management-brain/frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/goals/GoalDeviationChart.vue frontend/src/views/GoalsView.vue
git commit -m "feat: add KPI deviation chart for goal progress tracking"
```

---

### Task 8: Deploy + Verify

**Files:** None (deployment)

- [ ] **Step 1: Build frontend locally**

```bash
cd /Users/anna/Documents/ai-management-brain/frontend && npm run build
```

- [ ] **Step 2: Push to GitHub**

```bash
cd /Users/anna/Documents/ai-management-brain && git push
```

CI/CD will auto-deploy backend. Frontend needs manual rsync:

- [ ] **Step 3: Deploy frontend to server**

```bash
rsync -az --delete /Users/anna/Documents/ai-management-brain/frontend/dist/ ai-brain:~/ai-management-brain/frontend/dist/
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml up -d --build frontend'
```

- [ ] **Step 4: Apply migration on server**

```bash
ssh ai-brain 'cd ~/ai-management-brain && docker compose -f docker-compose.prod.yml exec api ./brain migrate up'
```

(Or however migrations are applied — check the existing migration approach.)

- [ ] **Step 5: Verify endpoints**

```bash
# Health check
ssh ai-brain 'curl -s localhost/healthz'

# Test goals API (needs auth token)
# Verify via frontend: navigate to /goals, create a goal, add key results
```

- [ ] **Step 6: Commit any deploy fixes**

If any fixes needed during deployment, commit them.
