-- name: ListMeetings :many
SELECT m.*, e.name AS employee_name, mg.name AS manager_name
FROM meetings m
JOIN employees e ON e.id = m.employee_id
LEFT JOIN employees mg ON mg.id = m.manager_id
WHERE m.tenant_id = $1
ORDER BY m.meeting_date DESC
LIMIT $2 OFFSET $3;

-- name: GetMeeting :one
SELECT m.*, e.name AS employee_name
FROM meetings m
JOIN employees e ON e.id = m.employee_id
WHERE m.id = $1 AND m.tenant_id = $2;

-- name: CreateMeeting :one
INSERT INTO meetings (tenant_id, employee_id, manager_id, meeting_date, duration_min, notes, mood, follow_up)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateMeeting :exec
UPDATE meetings
SET notes = $3, mood = $4, follow_up = $5, duration_min = $6, updated_at = now()
WHERE id = $1 AND tenant_id = $2;

-- name: DeleteMeeting :exec
DELETE FROM meetings WHERE id = $1 AND tenant_id = $2;

-- name: ListMeetingsByEmployee :many
SELECT m.*, e.name AS employee_name
FROM meetings m
JOIN employees e ON e.id = m.employee_id
WHERE m.employee_id = $1 AND m.tenant_id = $2
ORDER BY m.meeting_date DESC
LIMIT $3 OFFSET $4;

-- name: ListActionItems :many
SELECT ai.*, e.name AS assignee_name
FROM meeting_action_items ai
LEFT JOIN employees e ON e.id = ai.assignee_id
WHERE ai.meeting_id = $1
ORDER BY ai.created_at;

-- name: CreateActionItem :one
INSERT INTO meeting_action_items (meeting_id, title, assignee_id, due_date)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateActionItem :exec
UPDATE meeting_action_items
SET title = $2, status = $3, assignee_id = $4, due_date = $5, updated_at = now()
WHERE id = $1;

-- name: DeleteActionItem :exec
DELETE FROM meeting_action_items WHERE id = $1;

-- name: ListOpenActionItemsByTenant :many
SELECT ai.*, e.name AS assignee_name, m.meeting_date, emp.name AS employee_name
FROM meeting_action_items ai
JOIN meetings m ON m.id = ai.meeting_id
JOIN employees emp ON emp.id = m.employee_id
LEFT JOIN employees e ON e.id = ai.assignee_id
WHERE m.tenant_id = $1 AND ai.status != 'done'
ORDER BY ai.due_date NULLS LAST, ai.created_at;

-- name: VerifyMeetingTenant :one
SELECT tenant_id FROM meetings WHERE id = $1;
