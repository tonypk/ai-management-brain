-- name: CreateExecutionSignal :one
INSERT INTO execution_signals (tenant_id, subject_type, subject_id,
  signal_type, score, reasons, time_window)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListExecutionSignals :many
SELECT * FROM execution_signals
WHERE tenant_id = $1
  AND generated_at >= $2
ORDER BY score DESC, generated_at DESC
LIMIT $3;

-- name: ListSignalsByType :many
SELECT * FROM execution_signals
WHERE tenant_id = $1
  AND subject_type = $2
  AND generated_at >= $3
ORDER BY score DESC
LIMIT $4;

-- name: GetSignalsBySubject :many
SELECT * FROM execution_signals
WHERE subject_type = $1 AND subject_id = $2
ORDER BY generated_at DESC
LIMIT $3;

-- name: GetTopRisks :many
SELECT * FROM execution_signals
WHERE tenant_id = $1
  AND signal_type IN ('slow_response', 'missed_deadline', 'overloaded', 'blocker_risk', 'declining')
  AND generated_at >= $2
ORDER BY score DESC
LIMIT $3;

-- name: DeleteOldSignals :exec
DELETE FROM execution_signals
WHERE tenant_id = $1 AND generated_at < $2;
