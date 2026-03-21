-- name: CreateAIRoleInstance :one
INSERT INTO ai_role_instances (tenant_id, role_id, title, mentor_id, config)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, role_id) DO UPDATE SET
    title = EXCLUDED.title,
    mentor_id = EXCLUDED.mentor_id,
    config = EXCLUDED.config,
    is_active = true
RETURNING *;

-- name: ListActiveAIRoles :many
SELECT * FROM ai_role_instances
WHERE tenant_id = $1 AND is_active = true
ORDER BY created_at;

-- name: GetAIRoleInstance :one
SELECT * FROM ai_role_instances
WHERE tenant_id = $1 AND role_id = $2;

-- name: DeactivateAIRoles :exec
UPDATE ai_role_instances SET is_active = false
WHERE tenant_id = $1;

-- name: CreateAISuggestion :one
INSERT INTO ai_suggestions (tenant_id, role_id, role_title, capability, title, content, context_data)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListPendingSuggestions :many
SELECT * FROM ai_suggestions
WHERE tenant_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: ListRecentSuggestionsForTenant :many
SELECT * FROM ai_suggestions
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: UpdateSuggestionStatus :exec
UPDATE ai_suggestions SET status = $1, reviewed_at = now()
WHERE id = $2 AND tenant_id = $3;
