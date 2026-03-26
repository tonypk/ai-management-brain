-- name: CreateTask :one
INSERT INTO tasks (tenant_id, project_id, goal_id, key_result_id, title,
  description, owner_id, owner_team_id, status, priority, due_at,
  source_system, source_ref, created_by_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: ListTasks :many
SELECT t.*,
  e.name AS owner_name,
  p.name AS project_name
FROM tasks t
LEFT JOIN employees e ON t.owner_id = e.id
LEFT JOIN projects p ON t.project_id = p.id
WHERE t.tenant_id = $1
ORDER BY
  CASE t.priority
    WHEN 'critical' THEN 0
    WHEN 'high' THEN 1
    WHEN 'medium' THEN 2
    WHEN 'low' THEN 3
    ELSE 4
  END,
  t.due_at NULLS LAST;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1;

-- name: UpdateTask :one
UPDATE tasks SET
  title = $2, status = $3, priority = $4, owner_id = $5,
  due_at = $6, description = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: VerifyTaskTenant :one
SELECT tenant_id FROM tasks WHERE id = $1;

-- name: ListOverdueTasks :many
SELECT t.*, e.name AS owner_name
FROM tasks t
LEFT JOIN employees e ON t.owner_id = e.id
WHERE t.tenant_id = $1
  AND t.status NOT IN ('done', 'cancelled')
  AND t.due_at < now()
ORDER BY t.due_at;

-- name: ListTasksByOwner :many
SELECT * FROM tasks
WHERE owner_id = $1 AND status NOT IN ('done', 'cancelled')
ORDER BY due_at NULLS LAST;

-- name: CountTasksByStatus :many
SELECT status, COUNT(*)::int AS count
FROM tasks WHERE tenant_id = $1
GROUP BY status;
