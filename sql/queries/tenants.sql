-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: GetTenantByBossChatID :one
SELECT * FROM tenants WHERE boss_chat_id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (name, timezone, anthropic_key, mentor_id, bot_token, boss_chat_id, config)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateTenantMentor :exec
UPDATE tenants SET mentor_id = $2, mentor_blend = $3 WHERE id = $1;

-- name: UpdateTenantConfig :exec
UPDATE tenants SET config = $2 WHERE id = $1;

-- name: UpdateTenantNameTimezone :exec
UPDATE tenants SET name = $2, timezone = $3 WHERE id = $1;
