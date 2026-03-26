-- name: ListReviewCycles :many
SELECT * FROM review_cycles
WHERE tenant_id = $1
ORDER BY start_date DESC;

-- name: GetReviewCycle :one
SELECT * FROM review_cycles
WHERE id = $1 AND tenant_id = $2;

-- name: CreateReviewCycle :one
INSERT INTO review_cycles (tenant_id, title, period, status, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateReviewCycle :exec
UPDATE review_cycles
SET title = $3, status = $4, start_date = $5, end_date = $6, updated_at = now()
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteReviewCycle :exec
DELETE FROM review_cycles WHERE id = $1 AND tenant_id = $2;

-- name: ListReviewsByCycle :many
SELECT pr.*, e.name AS employee_name, r.name AS reviewer_name
FROM performance_reviews pr
JOIN employees e ON e.id = pr.employee_id
LEFT JOIN employees r ON r.id = pr.reviewer_id
WHERE pr.cycle_id = $1
ORDER BY e.name;

-- name: GetReview :one
SELECT pr.*, e.name AS employee_name
FROM performance_reviews pr
JOIN employees e ON e.id = pr.employee_id
WHERE pr.id = $1;

-- name: CreateReview :one
INSERT INTO performance_reviews (cycle_id, employee_id, reviewer_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateReview :exec
UPDATE performance_reviews
SET status = $2, self_rating = $3, manager_rating = $4,
    self_summary = $5, manager_summary = $6,
    strengths = $7, improvements = $8,
    submitted_at = CASE WHEN $2 = 'submitted' THEN now() ELSE submitted_at END,
    acknowledged_at = CASE WHEN $2 = 'acknowledged' THEN now() ELSE acknowledged_at END,
    updated_at = now()
WHERE id = $1;

-- name: DeleteReview :exec
DELETE FROM performance_reviews WHERE id = $1;

-- name: VerifyReviewCycleTenant :one
SELECT tenant_id FROM review_cycles WHERE id = $1;
