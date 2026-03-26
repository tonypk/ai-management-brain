-- name: ListTrainingPrograms :many
SELECT tp.*,
       COUNT(te.id) AS enrollment_count,
       COUNT(te.id) FILTER (WHERE te.status = 'completed') AS completed_count
FROM training_programs tp
LEFT JOIN training_enrollments te ON te.program_id = tp.id
WHERE tp.tenant_id = $1
GROUP BY tp.id
ORDER BY tp.created_at DESC;

-- name: GetTrainingProgram :one
SELECT * FROM training_programs WHERE id = $1;

-- name: CreateTrainingProgram :one
INSERT INTO training_programs (tenant_id, title, description, category, duration_hours, provider, url, is_mandatory)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateTrainingProgram :one
UPDATE training_programs SET
    title = $2,
    description = $3,
    category = $4,
    duration_hours = $5,
    provider = $6,
    url = $7,
    is_mandatory = $8,
    status = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTrainingProgram :exec
DELETE FROM training_programs WHERE id = $1;

-- name: VerifyTrainingProgramTenant :one
SELECT tenant_id FROM training_programs WHERE id = $1;

-- name: ListEnrollments :many
SELECT te.*, e.name AS employee_name
FROM training_enrollments te
JOIN employees e ON e.id = te.employee_id
WHERE te.program_id = $1
ORDER BY te.enrolled_at DESC;

-- name: CreateEnrollment :one
INSERT INTO training_enrollments (program_id, employee_id)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateEnrollment :one
UPDATE training_enrollments SET
    status = $2,
    completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END,
    score = $3,
    notes = $4
WHERE id = $1
RETURNING *;

-- name: DeleteEnrollment :exec
DELETE FROM training_enrollments WHERE id = $1;

-- name: ListEmployeeEnrollments :many
SELECT te.*, tp.title AS program_title, tp.category AS program_category
FROM training_enrollments te
JOIN training_programs tp ON tp.id = te.program_id
WHERE te.employee_id = $1
ORDER BY te.enrolled_at DESC;
