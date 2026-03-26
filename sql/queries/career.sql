-- name: ListCareerLevels :many
SELECT * FROM career_levels
WHERE tenant_id = $1
ORDER BY level_order ASC;

-- name: GetCareerLevel :one
SELECT * FROM career_levels WHERE id = $1;

-- name: CreateCareerLevel :one
INSERT INTO career_levels (tenant_id, title, level_order, description, requirements)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateCareerLevel :one
UPDATE career_levels SET
    title = $2,
    level_order = $3,
    description = $4,
    requirements = $5
WHERE id = $1
RETURNING *;

-- name: DeleteCareerLevel :exec
DELETE FROM career_levels WHERE id = $1;

-- name: VerifyCareerLevelTenant :one
SELECT tenant_id FROM career_levels WHERE id = $1;

-- name: ListCareerPaths :many
SELECT cp.*,
       e.name AS employee_name,
       cl_cur.title AS current_level_title,
       cl_cur.level_order AS current_level_order,
       cl_tgt.title AS target_level_title,
       cl_tgt.level_order AS target_level_order
FROM career_paths cp
JOIN employees e ON e.id = cp.employee_id
LEFT JOIN career_levels cl_cur ON cl_cur.id = cp.current_level_id
LEFT JOIN career_levels cl_tgt ON cl_tgt.id = cp.target_level_id
WHERE e.tenant_id = $1
ORDER BY e.name;

-- name: GetCareerPath :one
SELECT * FROM career_paths WHERE id = $1;

-- name: UpsertCareerPath :one
INSERT INTO career_paths (employee_id, current_level_id, target_level_id, target_date, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (employee_id) DO UPDATE SET
    current_level_id = EXCLUDED.current_level_id,
    target_level_id = EXCLUDED.target_level_id,
    target_date = EXCLUDED.target_date,
    notes = EXCLUDED.notes,
    updated_at = NOW()
RETURNING *;

-- name: DeleteCareerPath :exec
DELETE FROM career_paths WHERE id = $1;
