-- name: CreateChaseLog :one
INSERT INTO chase_logs (tenant_id, employee_id, report_date, step, action, message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetLastChaseStep :one
SELECT COALESCE(MAX(step), 0) as last_step FROM chase_logs
WHERE employee_id = $1 AND report_date = $2;
