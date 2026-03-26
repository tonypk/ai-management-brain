-- name: CreateWorkflow :one
INSERT INTO workflows (tenant_id, name, category, trigger_conditions, steps,
  approval_rules, escalation_rules)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListWorkflows :many
SELECT * FROM workflows
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;

-- name: GetWorkflow :one
SELECT * FROM workflows WHERE id = $1;

-- name: UpdateWorkflow :one
UPDATE workflows SET
  name = $2, category = $3, trigger_conditions = $4, steps = $5,
  approval_rules = $6, escalation_rules = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
UPDATE workflows SET is_active = false, updated_at = now() WHERE id = $1;

-- name: VerifyWorkflowTenant :one
SELECT tenant_id FROM workflows WHERE id = $1;
