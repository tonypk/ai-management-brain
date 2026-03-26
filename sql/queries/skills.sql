-- name: ListSkills :many
SELECT s.*, COUNT(es.id)::int AS employee_count
FROM skills s
LEFT JOIN employee_skills es ON es.skill_id = s.id
WHERE s.tenant_id = $1
GROUP BY s.id
ORDER BY s.category, s.name;

-- name: GetSkill :one
SELECT * FROM skills WHERE id = $1 AND tenant_id = $2;

-- name: CreateSkill :one
INSERT INTO skills (tenant_id, name, category, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSkill :exec
UPDATE skills
SET name = $3, category = $4, description = $5
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteSkill :exec
DELETE FROM skills WHERE id = $1 AND tenant_id = $2;

-- name: ListEmployeeSkills :many
SELECT es.*, s.name AS skill_name, s.category AS skill_category
FROM employee_skills es
JOIN skills s ON s.id = es.skill_id
WHERE es.employee_id = $1
ORDER BY s.category, s.name;

-- name: SetEmployeeSkill :one
INSERT INTO employee_skills (employee_id, skill_id, level, notes, assessed_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (employee_id, skill_id)
DO UPDATE SET level = EXCLUDED.level, notes = EXCLUDED.notes, assessed_at = now()
RETURNING *;

-- name: DeleteEmployeeSkill :exec
DELETE FROM employee_skills WHERE employee_id = $1 AND skill_id = $2;

-- name: GetSkillMatrix :many
SELECT e.id AS employee_id, e.name AS employee_name,
       s.id AS skill_id, s.name AS skill_name, s.category,
       COALESCE(es.level, 0)::int AS level
FROM employees e
CROSS JOIN skills s
LEFT JOIN employee_skills es ON es.employee_id = e.id AND es.skill_id = s.id
WHERE e.tenant_id = $1 AND s.tenant_id = $1 AND e.is_active = true
ORDER BY e.name, s.category, s.name;

-- name: VerifySkillTenant :one
SELECT tenant_id FROM skills WHERE id = $1;
