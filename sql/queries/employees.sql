-- name: GetEmployee :one
SELECT * FROM employees WHERE id = $1 AND tenant_id = $2;

-- name: GetEmployeeByTelegramID :one
SELECT * FROM employees WHERE telegram_id = $1;

-- name: ListActiveEmployees :many
SELECT * FROM employees WHERE tenant_id = $1 AND is_active = true ORDER BY name;

-- name: CreateEmployee :one
INSERT INTO employees (tenant_id, name, telegram_id, culture_code, role, invite_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateEmployeeTelegramID :exec
UPDATE employees SET telegram_id = $2 WHERE id = $1;

-- name: GetEmployeeByInviteCode :one
SELECT * FROM employees WHERE invite_code = $1 AND telegram_id IS NULL;

-- name: ListEmployeesWithoutReport :many
SELECT e.* FROM employees e
LEFT JOIN reports r ON e.id = r.employee_id AND r.report_date = $2
WHERE e.tenant_id = $1 AND e.is_active = true AND e.role = 'member' AND r.id IS NULL;
