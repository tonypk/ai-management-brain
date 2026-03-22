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

-- name: UpdateTenantChannels :exec
UPDATE tenants
SET slack_bot_token = $2, slack_signing_secret = $3,
    lark_app_id = $4, lark_app_secret = $5,
    signal_phone = $6, enabled_channels = $7
WHERE id = $1;

-- name: GetTenantChannelConfig :one
SELECT id, slack_bot_token, slack_signing_secret, lark_app_id, lark_app_secret, signal_phone, enabled_channels
FROM tenants WHERE id = $1;
