-- name: CreateRecommendation :one
INSERT INTO recommendations (
    tenant_id, category, priority, title, description,
    suggested_actions, evidence, source, target_entity_type,
    target_entity_id, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListRecommendations :many
SELECT * FROM recommendations
WHERE tenant_id = $1
  AND ($2::text = '' OR status = $2)
  AND ($3::text = '' OR category = $3)
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC
LIMIT $4 OFFSET $5;

-- name: GetRecommendation :one
SELECT * FROM recommendations
WHERE id = $1 AND tenant_id = $2;

-- name: GetRecommendationSummary :many
SELECT * FROM recommendations
WHERE tenant_id = $1 AND status = 'pending'
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC
LIMIT 3;

-- name: CountPendingRecommendations :one
SELECT count(*) FROM recommendations
WHERE tenant_id = $1 AND status = 'pending';

-- name: UpdateRecommendationStatus :exec
UPDATE recommendations
SET status = $3, reviewed_at = now(),
    executed_at = CASE WHEN $3 = 'executed' THEN now() ELSE executed_at END
WHERE id = $1 AND tenant_id = $2;

-- name: FindDuplicateRecommendation :one
SELECT id, priority FROM recommendations
WHERE tenant_id = $1
  AND category = $2
  AND target_entity_type IS NOT DISTINCT FROM $3
  AND target_entity_id IS NOT DISTINCT FROM $4
  AND status = 'pending'
  AND created_at > now() - interval '72 hours'
LIMIT 1;

-- name: FindDuplicateOrgRecommendation :one
SELECT id, priority FROM recommendations
WHERE tenant_id = $1
  AND category = $2
  AND target_entity_id IS NULL
  AND title = $3
  AND status = 'pending'
  AND created_at > now() - interval '72 hours'
LIMIT 1;

-- name: ExpireOldRecommendations :exec
UPDATE recommendations
SET status = 'expired'
WHERE tenant_id = $1
  AND status = 'pending'
  AND expires_at < now();

-- name: DeleteRecommendation :exec
DELETE FROM recommendations
WHERE id = $1 AND tenant_id = $2 AND status IN ('dismissed', 'expired');

-- name: ListActiveTenants :many
SELECT id, mentor_id FROM tenants;
