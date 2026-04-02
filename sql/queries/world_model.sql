-- sql/queries/world_model.sql

-- ===== Skills =====

-- name: UpsertWorldModelSkill :one
INSERT INTO world_model_skills (tenant_id, employee_id, skill_name, proficiency, source, confidence, mention_count)
VALUES ($1, $2, $3, $4, $5, $6, 1)
ON CONFLICT (tenant_id, employee_id, skill_name)
DO UPDATE SET
  proficiency = EXCLUDED.proficiency,
  confidence = GREATEST(world_model_skills.confidence, EXCLUDED.confidence),
  last_seen_at = now(),
  mention_count = world_model_skills.mention_count + 1
RETURNING *;

-- name: ListSkillsByTenant :many
SELECT s.*, e.name as employee_name
FROM world_model_skills s
JOIN employees e ON s.employee_id = e.id
WHERE s.tenant_id = $1 AND s.confidence >= 0.2
ORDER BY s.confidence DESC, s.mention_count DESC;

-- name: ListSkillsByEmployee :many
SELECT * FROM world_model_skills
WHERE tenant_id = $1 AND employee_id = $2 AND confidence >= 0.2
ORDER BY confidence DESC;

-- name: DecaySkillConfidence :exec
UPDATE world_model_skills
SET confidence = GREATEST(0.05,
  confidence * POWER(0.95, EXTRACT(EPOCH FROM (now() - last_seen_at)) / 604800.0)
  * LEAST(mention_count::numeric / 5.0, 2.0)
)
WHERE tenant_id = $1;

-- ===== Relationships =====

-- name: UpsertWorldModelRelationship :one
INSERT INTO world_model_relationships (tenant_id, employee_a_id, employee_b_id, relation_type, context, strength)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, employee_a_id, employee_b_id, relation_type)
DO UPDATE SET
  context = EXCLUDED.context,
  strength = LEAST(1.0, world_model_relationships.strength + 0.1),
  last_seen_at = now(),
  interaction_count = world_model_relationships.interaction_count + 1
RETURNING *;

-- name: ListRelationshipsByTenant :many
SELECT r.*, ea.name as employee_a_name, eb.name as employee_b_name
FROM world_model_relationships r
JOIN employees ea ON r.employee_a_id = ea.id
JOIN employees eb ON r.employee_b_id = eb.id
WHERE r.tenant_id = $1 AND r.strength >= 0.2
ORDER BY r.strength DESC;

-- name: ListRelationshipsByEmployee :many
SELECT r.*, ea.name as employee_a_name, eb.name as employee_b_name
FROM world_model_relationships r
JOIN employees ea ON r.employee_a_id = ea.id
JOIN employees eb ON r.employee_b_id = eb.id
WHERE r.tenant_id = $1 AND (r.employee_a_id = $2 OR r.employee_b_id = $2) AND r.strength >= 0.2
ORDER BY r.strength DESC;

-- name: DecayRelationshipStrength :exec
UPDATE world_model_relationships
SET strength = GREATEST(0.05,
  strength * POWER(0.95, EXTRACT(EPOCH FROM (now() - last_seen_at)) / 604800.0)
)
WHERE tenant_id = $1;

-- ===== Blockers =====

-- name: CreateWorldModelBlocker :one
INSERT INTO world_model_blockers (tenant_id, employee_id, category, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListActiveBlockersByTenant :many
SELECT b.*, e.name as employee_name
FROM world_model_blockers b
JOIN employees e ON b.employee_id = e.id
WHERE b.tenant_id = $1 AND b.status IN ('active', 'recurring')
ORDER BY b.recurrence_count DESC, b.first_seen_at DESC;

-- name: ListBlockersByEmployee :many
SELECT * FROM world_model_blockers
WHERE tenant_id = $1 AND employee_id = $2
ORDER BY first_seen_at DESC
LIMIT 20;

-- name: ResolveBlocker :exec
UPDATE world_model_blockers
SET status = 'resolved', resolved_at = now()
WHERE id = $1;

-- name: IncrementBlockerRecurrence :exec
UPDATE world_model_blockers
SET recurrence_count = recurrence_count + 1, status = 'recurring', resolved_at = NULL
WHERE id = $1;

-- name: FindSimilarBlocker :one
SELECT * FROM world_model_blockers
WHERE tenant_id = $1 AND employee_id = $2 AND category = $3 AND status != 'resolved'
ORDER BY first_seen_at DESC
LIMIT 1;

-- ===== Growth Events =====

-- name: CreateGrowthEvent :one
INSERT INTO world_model_growth_events (tenant_id, employee_id, event_type, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListGrowthEventsByTenant :many
SELECT g.*, e.name as employee_name
FROM world_model_growth_events g
JOIN employees e ON g.employee_id = e.id
WHERE g.tenant_id = $1
ORDER BY g.detected_at DESC
LIMIT 50;

-- name: ListGrowthEventsByEmployee :many
SELECT * FROM world_model_growth_events
WHERE tenant_id = $1 AND employee_id = $2
ORDER BY detected_at DESC;

-- ===== Insights =====

-- name: CreateWorldModelInsight :one
INSERT INTO world_model_insights (tenant_id, dimension, insight_text, evidence, confidence, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListActiveInsightsByTenant :many
SELECT * FROM world_model_insights
WHERE tenant_id = $1 AND expires_at > now()
ORDER BY confidence DESC, generated_at DESC;

-- name: ExpireOldInsights :exec
DELETE FROM world_model_insights
WHERE expires_at < now();

-- ===== Overview Aggregation =====

-- name: CountSkillsByTenant :one
SELECT COUNT(DISTINCT skill_name) FROM world_model_skills
WHERE tenant_id = $1 AND confidence >= 0.2;

-- name: CountRelationshipsByTenant :one
SELECT COUNT(*) FROM world_model_relationships
WHERE tenant_id = $1 AND strength >= 0.2;

-- name: CountActiveBlockersByTenant :one
SELECT COUNT(*) FROM world_model_blockers
WHERE tenant_id = $1 AND status IN ('active', 'recurring');

-- name: CountGrowthEventsByTenant :one
SELECT COUNT(*) FROM world_model_growth_events
WHERE tenant_id = $1 AND detected_at > now() - INTERVAL '30 days';

-- name: GetBlockerCategoryBreakdown :many
SELECT category, COUNT(*) as count
FROM world_model_blockers
WHERE tenant_id = $1 AND status IN ('active', 'recurring')
GROUP BY category
ORDER BY count DESC;
