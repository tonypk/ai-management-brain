-- name: CreateEngagement :one
INSERT INTO engagements (tenant_id, title, problem_statement, tier, category, phase,
  diagnosis_data, mentor_id, culture_code, next_check_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetEngagement :one
SELECT * FROM engagements WHERE id = $1;

-- name: ListEngagementsByTenant :many
SELECT * FROM engagements
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveEngagements :many
SELECT * FROM engagements
WHERE tenant_id = $1 AND phase NOT IN ('closed')
ORDER BY updated_at DESC;

-- name: ListEngagementsForTracking :many
SELECT * FROM engagements
WHERE phase IN ('executing', 'tracking')
  AND (next_check_at IS NULL OR next_check_at <= now());

-- name: UpdateEngagementPhase :exec
UPDATE engagements SET phase = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementDiagnosis :exec
UPDATE engagements SET
  diagnosis_questions = $2, diagnosis_answers = $3, updated_at = now()
WHERE id = $1;

-- name: UpdateEngagementAnalysis :exec
UPDATE engagements SET analysis = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementPlan :exec
UPDATE engagements SET plan = $2, updated_at = now() WHERE id = $1;

-- name: UpdateEngagementProgress :exec
UPDATE engagements SET
  progress_pct = $2, next_check_at = $3, updated_at = now()
WHERE id = $1;

-- name: CloseEngagement :exec
UPDATE engagements SET phase = 'closed', closed_at = now(), updated_at = now()
WHERE id = $1;

-- name: VerifyEngagementTenant :one
SELECT tenant_id FROM engagements WHERE id = $1;

-- name: CreateEngagementAction :one
INSERT INTO engagement_actions (engagement_id, action_type, title, description,
  params, owner_name, priority, due_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListEngagementActions :many
SELECT * FROM engagement_actions
WHERE engagement_id = $1
ORDER BY
  CASE priority
    WHEN 'critical' THEN 0
    WHEN 'high' THEN 1
    WHEN 'medium' THEN 2
    WHEN 'low' THEN 3
    ELSE 4
  END,
  created_at;

-- name: GetEngagementAction :one
SELECT * FROM engagement_actions WHERE id = $1;

-- name: ApproveEngagementAction :exec
UPDATE engagement_actions SET status = 'approved', approved_at = now(), updated_at = now()
WHERE id = $1;

-- name: RejectEngagementAction :exec
UPDATE engagement_actions SET status = 'rejected', updated_at = now() WHERE id = $1;

-- name: MarkEngagementActionDone :exec
UPDATE engagement_actions SET
  status = 'done', executed_at = now(), result = $2, updated_at = now()
WHERE id = $1;

-- name: MarkEngagementActionFailed :exec
UPDATE engagement_actions SET
  status = 'failed', result = $2, updated_at = now()
WHERE id = $1;

-- name: LinkEngagementActionTask :exec
UPDATE engagement_actions SET linked_task_id = $2, updated_at = now() WHERE id = $1;

-- name: LinkEngagementActionMeeting :exec
UPDATE engagement_actions SET linked_meeting_id = $2, updated_at = now() WHERE id = $1;

-- name: CountEngagementActionsByStatus :many
SELECT status, COUNT(*)::int AS count
FROM engagement_actions WHERE engagement_id = $1
GROUP BY status;

-- name: ListApprovedEngagementActions :many
SELECT * FROM engagement_actions
WHERE engagement_id = $1 AND status = 'approved'
ORDER BY created_at;

-- name: ListEngagementActionsWithLinks :many
SELECT ea.*,
  t.status AS task_status,
  m.mood AS meeting_mood
FROM engagement_actions ea
LEFT JOIN tasks t ON ea.linked_task_id = t.id
LEFT JOIN meetings m ON ea.linked_meeting_id = m.id
WHERE ea.engagement_id = $1
ORDER BY ea.created_at;
