-- name: CreateGroupChat :one
INSERT INTO group_chats (tenant_id, platform, platform_chat_id, name, group_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetGroupChatByID :one
SELECT * FROM group_chats
WHERE id = $1;

-- name: GetGroupChatByPlatformID :one
SELECT * FROM group_chats
WHERE platform = $1 AND platform_chat_id = $2;

-- name: ListActiveGroupChatsByTenant :many
SELECT * FROM group_chats
WHERE tenant_id = $1 AND is_active = true
ORDER BY created_at;

-- name: ListGroupChatsByTenant :many
SELECT * FROM group_chats
WHERE tenant_id = $1
ORDER BY created_at;

-- name: UpdateGroupChat :one
UPDATE group_chats
SET name = $2, group_type = $3, is_active = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteGroupChat :exec
UPDATE group_chats
SET is_active = false, updated_at = now()
WHERE id = $1;
