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
