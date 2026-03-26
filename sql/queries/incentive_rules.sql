-- name: CreateIncentiveRule :one
INSERT INTO incentive_rules (tenant_id, name, reward_model, payout_cycle,
  attribution_rules, penalty_rules, scoring_formula, applies_to)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListIncentiveRules :many
SELECT * FROM incentive_rules
WHERE tenant_id = $1 AND is_active = true
ORDER BY name;

-- name: GetIncentiveRule :one
SELECT * FROM incentive_rules WHERE id = $1;

-- name: UpdateIncentiveRule :one
UPDATE incentive_rules SET
  name = $2, reward_model = $3, payout_cycle = $4,
  attribution_rules = $5, scoring_formula = $6,
  applies_to = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteIncentiveRule :exec
UPDATE incentive_rules SET is_active = false, updated_at = now() WHERE id = $1;

-- name: VerifyIncentiveRuleTenant :one
SELECT tenant_id FROM incentive_rules WHERE id = $1;

-- name: CreateIncentiveScore :one
INSERT INTO incentive_scores (tenant_id, rule_id, person_id, period,
  score, score_breakdown, payout_weight, attribution_confidence, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (rule_id, person_id, period)
DO UPDATE SET
  score = EXCLUDED.score,
  score_breakdown = EXCLUDED.score_breakdown,
  payout_weight = EXCLUDED.payout_weight,
  attribution_confidence = EXCLUDED.attribution_confidence,
  calculated_at = now()
RETURNING *;

-- name: ListIncentiveScores :many
SELECT ics.*, e.name AS person_name, ir.name AS rule_name
FROM incentive_scores ics
JOIN employees e ON ics.person_id = e.id
JOIN incentive_rules ir ON ics.rule_id = ir.id
WHERE ics.tenant_id = $1 AND ics.period = $2
ORDER BY ics.score DESC;

-- name: GetIncentiveScoresByPerson :many
SELECT ics.*, ir.name AS rule_name
FROM incentive_scores ics
JOIN incentive_rules ir ON ics.rule_id = ir.id
WHERE ics.person_id = $1
ORDER BY ics.period DESC, ir.name;
