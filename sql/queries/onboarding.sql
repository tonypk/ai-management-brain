-- name: CreateOnboardingSession :one
INSERT INTO onboarding_sessions (tenant_id, status, channel_type)
VALUES ($1, 'onboarding', $2)
RETURNING *;

-- name: GetOnboardingSession :one
SELECT * FROM onboarding_sessions WHERE tenant_id = $1;

-- name: UpdateOnboardingSession :exec
UPDATE onboarding_sessions
SET status = $2, confirm_step = $3, collected_data = $4,
    proposed_plan = $5, message_count = $6, channel_type = $7,
    updated_at = now()
WHERE tenant_id = $1;

-- name: DeleteOnboardingSession :exec
DELETE FROM onboarding_sessions WHERE tenant_id = $1;

-- name: GetTenantByBossSlackID :one
SELECT * FROM tenants WHERE boss_slack_id = $1 AND boss_slack_id IS NOT NULL;

-- name: GetTenantByBossLarkID :one
SELECT * FROM tenants WHERE boss_lark_id = $1 AND boss_lark_id IS NOT NULL;

-- name: UpdateOrganizationFromOnboarding :exec
UPDATE organizations
SET industry = $2, size = $3, stage = $4, business_model = $5,
    management_pain_points = $6, current_projects = $7,
    target_framework = $8, team_structure = $9,
    communication_tools = $10, culture_preferences = $11,
    updated_at = now()
WHERE tenant_id = $1;

-- name: SetTenantOnboardingCompleted :exec
UPDATE tenants SET onboarding_completed_at = now() WHERE id = $1;
