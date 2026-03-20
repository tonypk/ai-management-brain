-- name: CreateReport :one
INSERT INTO reports (tenant_id, employee_id, report_date, answers, blockers, sentiment)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetReportsByTenantDate :many
SELECT r.*, e.name as employee_name FROM reports r
JOIN employees e ON r.employee_id = e.id
WHERE r.tenant_id = $1 AND r.report_date = $2
ORDER BY r.submitted_at;

-- name: CountReportsByTenantDate :one
SELECT COUNT(*) FROM reports WHERE tenant_id = $1 AND report_date = $2;

-- name: UpdateReportAnalysis :exec
UPDATE reports SET blockers = $2, sentiment = $3
WHERE id = $1;

-- name: GetLatestReportByEmployee :one
SELECT * FROM reports
WHERE employee_id = $1 AND report_date = $2
ORDER BY submitted_at DESC LIMIT 1;

-- name: GetEmployeeSubmissionHistory :many
SELECT report_date, sentiment FROM reports
WHERE employee_id = $1 AND report_date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY report_date DESC;

-- name: GetConsecutiveMissDays :one
SELECT COUNT(*) as missed_days FROM generate_series(
    CURRENT_DATE - INTERVAL '7 days', CURRENT_DATE - INTERVAL '1 day', '1 day'
) d(day)
WHERE d.day::date NOT IN (
    SELECT report_date FROM reports WHERE employee_id = $1 AND report_date >= CURRENT_DATE - INTERVAL '7 days'
)
AND d.day::date >= (
    COALESCE(
        (SELECT MAX(report_date) FROM reports WHERE employee_id = $1 AND report_date >= CURRENT_DATE - INTERVAL '7 days'),
        CURRENT_DATE - INTERVAL '7 days'
    )
);

-- name: GetRecentSentiments :many
SELECT sentiment FROM reports
WHERE employee_id = $1 AND sentiment IS NOT NULL
AND report_date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY report_date DESC
LIMIT $2;

-- name: GetEmployeeReportStreak :one
SELECT COUNT(*) as missed_days FROM generate_series(
    CURRENT_DATE - INTERVAL '7 days', CURRENT_DATE - INTERVAL '1 day', '1 day'
) d(day)
WHERE NOT EXISTS (
    SELECT 1 FROM reports WHERE employee_id = $1 AND report_date = d.day::date
);
