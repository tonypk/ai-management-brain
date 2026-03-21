-- sql/queries/memories.sql

-- name: CreateMemory :one
INSERT INTO memories (
    tenant_id, memory_type, memory_tier, employee_id, source_type, source_id,
    content, summary, embedding, importance, metadata, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetMemory :one
SELECT * FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: ListMemoriesByTenant :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND ($2::varchar = '' OR memory_type = $2)
  AND ($3::varchar = '' OR memory_tier = $3)
  AND ($4::varchar = '' OR employee_id::text = $4)
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;

-- name: CountMemoriesByTenant :one
SELECT COUNT(*) FROM memories
WHERE tenant_id = $1 AND merged_into IS NULL;

-- name: ListShortTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'short_term'
  AND merged_into IS NULL
ORDER BY created_at ASC
LIMIT 200;

-- name: ListLongTermByEmployee :many
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'long_term'
  AND merged_into IS NULL
ORDER BY importance DESC, created_at DESC;

-- name: GetProfileByEmployee :one
SELECT * FROM memories
WHERE tenant_id = $1
  AND employee_id = $2
  AND memory_tier = 'profile'
  AND merged_into IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateMemoryMergedInto :exec
UPDATE memories SET merged_into = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteExpiredMemories :execrows
DELETE FROM memories WHERE expires_at < NOW() AND merged_into IS NULL;

-- name: DeleteMemory :exec
DELETE FROM memories WHERE id = $1 AND tenant_id = $2;

-- name: BackfillEmbedding :exec
UPDATE memories SET embedding = $2, updated_at = NOW()
WHERE id = $1 AND embedding IS NULL;

-- name: IncrementAccessCount :exec
UPDATE memories SET access_count = access_count + 1, updated_at = NOW() WHERE id = $1;

-- name: ListTenantsWithMemories :many
SELECT DISTINCT tenant_id FROM memories WHERE merged_into IS NULL;

-- name: ListEmployeesWithShortTermMemories :many
SELECT DISTINCT employee_id FROM memories
WHERE tenant_id = $1
  AND memory_tier = 'short_term'
  AND merged_into IS NULL
  AND employee_id IS NOT NULL;

-- name: ListEmployeesWithLongTermMemories :many
SELECT DISTINCT employee_id FROM memories
WHERE tenant_id = $1
  AND memory_tier = 'long_term'
  AND merged_into IS NULL
  AND employee_id IS NOT NULL;
