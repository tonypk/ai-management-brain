-- name: CreateOrgUnit :one
INSERT INTO org_units (tenant_id, parent_id, name, unit_type, head_role, responsibilities, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListOrgUnits :many
SELECT * FROM org_units WHERE tenant_id = $1 AND is_active = true ORDER BY sort_order, name;

-- name: GetOrgUnit :one
SELECT * FROM org_units WHERE id = $1;

-- name: UpdateOrgUnit :exec
UPDATE org_units
SET name = $2, unit_type = $3, parent_id = $4, head_role = $5,
    head_employee_id = $6, responsibilities = $7, updated_at = now()
WHERE id = $1;

-- name: SoftDeleteOrgUnit :exec
UPDATE org_units SET is_active = false, updated_at = now() WHERE id = $1;

-- name: DeleteOrgUnitsByTenant :exec
DELETE FROM org_units WHERE tenant_id = $1;

-- name: AssignEmployeeToUnit :exec
UPDATE employees SET org_unit_id = $2 WHERE id = $1;

-- name: ListEmployeesByUnit :many
SELECT * FROM employees WHERE org_unit_id = $1 AND is_active = true ORDER BY name;
