-- name: CreateWizardSession :one
INSERT INTO wizard_sessions (tenant_id, mentor_id, current_step, conversation, company_profile)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetLatestWizardSession :one
SELECT * FROM wizard_sessions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: UpdateWizardSession :exec
UPDATE wizard_sessions SET current_step = $2, conversation = $3, company_profile = $4, updated_at = now() WHERE id = $1;

-- name: DeleteWizardSessions :exec
DELETE FROM wizard_sessions WHERE tenant_id = $1;
