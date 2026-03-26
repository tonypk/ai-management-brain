-- name: CreateMetric :one
INSERT INTO metrics (tenant_id, name, display_name, formula, unit, source,
  refresh_frequency, target_value, alert_threshold, owner_id, owner_team_id, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: ListMetrics :many
SELECT * FROM metrics
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;

-- name: GetMetric :one
SELECT * FROM metrics WHERE id = $1;

-- name: UpdateMetric :one
UPDATE metrics SET
  display_name = $2, formula = $3, unit = $4, source = $5,
  target_value = $6, alert_threshold = $7, owner_id = $8,
  tags = $9, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteMetric :exec
UPDATE metrics SET is_active = false, updated_at = now() WHERE id = $1;

-- name: VerifyMetricTenant :one
SELECT tenant_id FROM metrics WHERE id = $1;

-- name: IngestMetricValue :one
INSERT INTO metric_values (metric_id, observed_at, value, dimensions, source_ref)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListMetricValues :many
SELECT * FROM metric_values
WHERE metric_id = $1 AND observed_at >= $2
ORDER BY observed_at DESC
LIMIT $3;

-- name: GetLatestMetricValue :one
SELECT * FROM metric_values
WHERE metric_id = $1
ORDER BY observed_at DESC
LIMIT 1;

-- name: GetMetricsWithLatestValues :many
SELECT m.id, m.tenant_id, m.name, m.display_name, m.formula, m.unit,
  m.source, m.refresh_frequency, m.target_value, m.alert_threshold,
  m.owner_id, m.owner_team_id, m.tags, m.is_active, m.created_at, m.updated_at,
  mv.value AS latest_value,
  mv.observed_at AS latest_observed_at
FROM metrics m
LEFT JOIN LATERAL (
  SELECT value, observed_at
  FROM metric_values WHERE metric_id = m.id
  ORDER BY observed_at DESC LIMIT 1
) mv ON true
WHERE m.tenant_id = $1 AND m.is_active = true
ORDER BY m.name;
