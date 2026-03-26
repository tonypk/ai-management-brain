-- name: CreateProject :one
INSERT INTO projects (tenant_id, name, description, owner_id, owner_team_id,
  status, priority, linked_goal_ids, linked_metric_ids, source_system, source_ref,
  start_date, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: ListProjects :many
SELECT p.*,
  e.name AS owner_name,
  ou.name AS team_name
FROM projects p
LEFT JOIN employees e ON p.owner_id = e.id
LEFT JOIN org_units ou ON p.owner_team_id = ou.id
WHERE p.tenant_id = $1
ORDER BY p.created_at DESC;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: UpdateProject :one
UPDATE projects SET
  name = $2, description = $3, status = $4, priority = $5,
  owner_id = $6, blockers = $7, due_date = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: VerifyProjectTenant :one
SELECT tenant_id FROM projects WHERE id = $1;

-- name: ListBlockedProjects :many
SELECT * FROM projects
WHERE tenant_id = $1 AND status = 'blocked'
ORDER BY priority, created_at;
