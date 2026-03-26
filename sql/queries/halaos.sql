-- name: GetHalaOSLinkByCompanyID :one
SELECT * FROM halaos_links
WHERE halaos_company_id = $1 AND is_active = true
LIMIT 1;

-- name: GetHalaOSLinkByTenant :one
SELECT * FROM halaos_links
WHERE tenant_id = $1 AND is_active = true
LIMIT 1;

-- name: CreateHalaOSLink :one
INSERT INTO halaos_links (tenant_id, webhook_secret, halaos_company_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteHalaOSLink :exec
DELETE FROM halaos_links WHERE id = $1 AND tenant_id = $2;

-- name: GetHalaOSEventByKey :one
SELECT id FROM halaos_events
WHERE tenant_id = $1 AND idempotency_key = $2
LIMIT 1;

-- name: CreateHalaOSEvent :one
INSERT INTO halaos_events (tenant_id, event_type, idempotency_key, payload)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEmployeeByHalaOSID :one
SELECT * FROM employees
WHERE tenant_id = $1 AND halaos_employee_id = $2
LIMIT 1;

-- name: GetEmployeeByHalaOSNo :one
SELECT * FROM employees
WHERE tenant_id = $1 AND halaos_employee_no = $2
LIMIT 1;

-- name: CreateEmployeeFromHalaOS :one
INSERT INTO employees (tenant_id, name, role, halaos_employee_id, halaos_employee_no)
VALUES ($1, $2, 'member', $3, $4)
RETURNING *;

-- name: UpdateEmployeeHalaOSLink :exec
UPDATE employees
SET halaos_employee_id = $2, halaos_employee_no = $3
WHERE id = $1;

-- name: CountHRSignalsByType :many
SELECT signal_type, COUNT(*) AS count
FROM execution_signals
WHERE tenant_id = $1
  AND signal_type IN ('flight_risk', 'burnout_risk', 'team_health', 'org_health')
  AND generated_at >= $2
GROUP BY signal_type;

-- name: CountHighRiskSignals :one
SELECT COUNT(*) FROM execution_signals
WHERE tenant_id = $1
  AND signal_type = $2
  AND score >= $3::numeric
  AND generated_at >= $4;

-- name: GetLatestOrgHealthSignal :one
SELECT * FROM execution_signals
WHERE tenant_id = $1
  AND signal_type = 'org_health'
  AND subject_type = 'organization'
ORDER BY generated_at DESC
LIMIT 1;

-- name: CountRecentCommunicationEvents :one
SELECT COUNT(*) FROM communication_events
WHERE tenant_id = $1
  AND source_type = 'halaos'
  AND event_type = $2
  AND occurred_at >= $3;
