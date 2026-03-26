-- name: CreateCommunicationEvent :one
INSERT INTO communication_events (tenant_id, source_type, source_id, platform,
  event_type, actor_id, target_id, related_task_id, related_project_id,
  related_goal_id, payload, confidence, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: ListCommunicationEvents :many
SELECT ce.*,
  actor.name AS actor_name,
  target.name AS target_name
FROM communication_events ce
LEFT JOIN employees actor ON ce.actor_id = actor.id
LEFT JOIN employees target ON ce.target_id = target.id
WHERE ce.tenant_id = $1
  AND ce.occurred_at >= $2
ORDER BY ce.occurred_at DESC
LIMIT $3;

-- name: ListEventsByType :many
SELECT ce.*,
  actor.name AS actor_name,
  target.name AS target_name
FROM communication_events ce
LEFT JOIN employees actor ON ce.actor_id = actor.id
LEFT JOIN employees target ON ce.target_id = target.id
WHERE ce.tenant_id = $1
  AND ce.event_type = $2
  AND ce.occurred_at >= $3
ORDER BY ce.occurred_at DESC
LIMIT $4;

-- name: ListEventsByPerson :many
SELECT * FROM communication_events
WHERE tenant_id = $1
  AND (actor_id = $2 OR target_id = $2)
  AND occurred_at >= $3
ORDER BY occurred_at DESC;

-- name: CountEventsByType :many
SELECT event_type, COUNT(*)::int AS count
FROM communication_events
WHERE tenant_id = $1 AND occurred_at >= $2
GROUP BY event_type
ORDER BY count DESC;
