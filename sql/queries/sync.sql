-- name: CreateSyncConfig :one
INSERT INTO sync_configs (tenant_id, storage_type, is_enabled, entity_types, sync_frequency_minutes, config)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, storage_type) DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  entity_types = EXCLUDED.entity_types,
  sync_frequency_minutes = EXCLUDED.sync_frequency_minutes,
  config = EXCLUDED.config,
  updated_at = now()
RETURNING *;

-- name: GetSyncConfig :one
SELECT * FROM sync_configs WHERE tenant_id = $1 AND storage_type = $2;

-- name: ListSyncConfigs :many
SELECT * FROM sync_configs WHERE tenant_id = $1 ORDER BY storage_type;

-- name: UpdateLastSync :exec
UPDATE sync_configs SET last_sync_at = $2, last_sync_status = $3, updated_at = now()
WHERE id = $1;

-- name: CreateSyncLog :one
INSERT INTO sync_logs (tenant_id, sync_config_id, direction)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CompleteSyncLog :exec
UPDATE sync_logs SET
  completed_at = now(),
  status = $2,
  items_pushed = $3,
  items_pulled = $4,
  conflicts = $5,
  errors = $6,
  summary = $7
WHERE id = $1;

-- name: ListSyncLogs :many
SELECT * FROM sync_logs WHERE sync_config_id = $1 ORDER BY started_at DESC LIMIT $2;

-- name: GetChangedTasks :many
SELECT id, title, status, priority, assignee_name, due_date, external_id, external_source, updated_at
FROM tasks
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedGoals :many
SELECT id, title, level, goal_type, status, external_id, external_source, updated_at
FROM goals
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedProjects :many
SELECT id, name, status, priority, external_id, external_source, updated_at
FROM projects
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;

-- name: GetChangedMetrics :many
SELECT id, name, unit, current_value, target_value, external_id, external_source, updated_at
FROM metrics
WHERE tenant_id = $1 AND updated_at > $2
ORDER BY updated_at DESC;
