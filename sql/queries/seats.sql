-- name: ListSeatsByTenant :many
SELECT * FROM seats WHERE tenant_id = $1 ORDER BY seat_type;

-- name: ListActiveSeatsByTenant :many
SELECT * FROM seats WHERE tenant_id = $1 AND is_active = true ORDER BY seat_type;

-- name: GetSeatByType :one
SELECT * FROM seats WHERE tenant_id = $1 AND seat_type = $2;

-- name: GetSeatByID :one
SELECT * FROM seats WHERE id = $1;

-- name: CreateSeat :one
INSERT INTO seats (tenant_id, seat_type, title, persona_id, scope)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateSeat :one
UPDATE seats SET title = $2, persona_id = $3, scope = $4, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteSeat :exec
DELETE FROM seats WHERE id = $1;
