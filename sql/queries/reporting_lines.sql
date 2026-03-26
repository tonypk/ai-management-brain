-- name: CreateReportingLine :one
INSERT INTO reporting_lines (tenant_id, manager_id, report_id, relationship_type)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListReportingLines :many
SELECT rl.*,
  mgr.name AS manager_name,
  rpt.name AS report_name
FROM reporting_lines rl
JOIN employees mgr ON rl.manager_id = mgr.id
JOIN employees rpt ON rl.report_id = rpt.id
WHERE rl.tenant_id = $1
ORDER BY mgr.name, rpt.name;

-- name: GetDirectReports :many
SELECT e.* FROM employees e
JOIN reporting_lines rl ON rl.report_id = e.id
WHERE rl.manager_id = $1 AND rl.relationship_type = 'direct'
ORDER BY e.name;

-- name: GetManager :one
SELECT e.* FROM employees e
JOIN reporting_lines rl ON rl.manager_id = e.id
WHERE rl.report_id = $1 AND rl.relationship_type = 'direct'
LIMIT 1;

-- name: DeleteReportingLine :exec
DELETE FROM reporting_lines WHERE id = $1;

-- name: VerifyReportingLineTenant :one
SELECT tenant_id FROM reporting_lines WHERE id = $1;
