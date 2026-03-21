-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, tenant_id, prefix, key_hash, name, scopes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, tenant_id, prefix, key_hash, name, scopes, is_active, last_used_at, created_at;

-- name: GetAPIKeyByHash :one
SELECT ak.id, ak.user_id, ak.tenant_id, ak.prefix, ak.key_hash, ak.name, ak.scopes, ak.is_active, ak.last_used_at, ak.created_at,
       u.role
FROM api_keys ak
JOIN users u ON u.id = ak.user_id AND u.is_active = true
WHERE ak.key_hash = $1 AND ak.is_active = true;

-- name: ListAPIKeysByUser :many
SELECT id, user_id, tenant_id, prefix, key_hash, name, scopes, is_active, last_used_at, created_at
FROM api_keys
WHERE user_id = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: RevokeAPIKey :exec
UPDATE api_keys SET is_active = false WHERE id = $1 AND user_id = $2;

-- name: TouchAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;

-- name: GetTenantPlan :one
SELECT plan FROM tenants WHERE id = $1;
