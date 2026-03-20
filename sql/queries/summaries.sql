-- name: CreateSummary :one
INSERT INTO summaries (tenant_id, summary_date, content, submission_rate, blockers_count, key_metrics)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSummary :one
SELECT * FROM summaries WHERE tenant_id = $1 AND summary_date = $2;

-- name: GetLatestSummary :one
SELECT * FROM summaries WHERE tenant_id = $1 ORDER BY summary_date DESC LIMIT 1;
