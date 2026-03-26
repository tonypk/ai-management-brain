-- name: CreateWorkingMemorySnapshot :one
INSERT INTO working_memory_snapshots (tenant_id, snapshot_type, content, generated_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLatestSnapshot :one
SELECT * FROM working_memory_snapshots
WHERE tenant_id = $1 AND snapshot_type = $2
ORDER BY generated_at DESC
LIMIT 1;

-- name: ListSnapshots :many
SELECT * FROM working_memory_snapshots
WHERE tenant_id = $1
  AND generated_at >= $2
ORDER BY generated_at DESC
LIMIT $3;

-- name: DeleteOldSnapshots :exec
DELETE FROM working_memory_snapshots
WHERE tenant_id = $1 AND generated_at < $2;
