-- name: GetEmployee :one
SELECT * FROM employees WHERE id = $1 AND tenant_id = $2;

-- name: GetEmployeeByTelegramID :one
SELECT * FROM employees WHERE telegram_id = $1;

-- name: ListActiveEmployees :many
SELECT * FROM employees WHERE tenant_id = $1 AND is_active = true ORDER BY name;

-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code, job_title, responsibilities, country, language)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateEmployeeTelegramID :exec
UPDATE employees SET telegram_id = $2 WHERE id = $1;

-- name: GetEmployeeByInviteCode :one
SELECT * FROM employees WHERE invite_code = $1 AND telegram_id IS NULL;

-- name: UpdateEmployeeCulture :exec
UPDATE employees SET culture_code = $2 WHERE id = $1;

-- name: ListEmployeesWithoutReport :many
SELECT e.* FROM employees e
LEFT JOIN reports r ON e.id = r.employee_id AND r.report_date = $2
WHERE e.tenant_id = $1 AND e.is_active = true AND e.role = 'member' AND r.id IS NULL;

-- name: GetEmployeeBySignalPhone :one
SELECT * FROM employees WHERE signal_phone = $1 AND is_active = true;

-- name: GetEmployeeBySlackID :one
SELECT * FROM employees WHERE slack_id = $1 AND is_active = true;

-- name: GetEmployeeByLarkID :one
SELECT * FROM employees WHERE lark_id = $1 AND is_active = true;

-- name: UpdateEmployeeChannels :exec
UPDATE employees
SET signal_phone = $2, slack_id = $3, lark_id = $4, preferred_channel = $5
WHERE id = $1;

-- name: UpdateEmployeePreferredChannel :exec
UPDATE employees SET preferred_channel = $2 WHERE id = $1;

-- name: UpdateEmployeeProfile :exec
UPDATE employees
SET job_title = $2, responsibilities = $3, country = $4, language = $5
WHERE id = $1;

-- name: ListEmployeesWithChannels :many
SELECT id, tenant_id, name, telegram_id, signal_phone, slack_id, lark_id, preferred_channel, culture_code, role, is_active, job_title, responsibilities, country, language
FROM employees
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;

-- name: GetEmployeeByNameFuzzy :one
SELECT * FROM employees
WHERE tenant_id = $1 AND is_active = true AND name ILIKE '%' || $2 || '%'
ORDER BY name
LIMIT 1;
