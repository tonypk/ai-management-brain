-- name: CreateOrganization :one
INSERT INTO organizations (tenant_id, industry, size, stage, business_model, region, mentor_id, management_plan, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: GetOrganizationByTenant :one
SELECT * FROM organizations WHERE tenant_id = $1;

-- name: UpdateOrganizationPlan :exec
UPDATE organizations SET management_plan = $2, plan_version = plan_version + 1, updated_at = now() WHERE tenant_id = $1;

-- name: UpdateOrganizationStatus :exec
UPDATE organizations SET status = $2, updated_at = now() WHERE tenant_id = $1;

-- name: DeleteOrganization :exec
DELETE FROM organizations WHERE tenant_id = $1;

-- name: UpsertOrganization :one
INSERT INTO organizations (
    tenant_id, industry, size, stage, business_model, mentor_id,
    management_plan, status, management_pain_points, current_projects,
    target_framework, team_structure, communication_tools, culture_preferences
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'draft', $8, $9, $10, $11, $12, $13
)
ON CONFLICT (tenant_id) DO UPDATE SET
    industry = EXCLUDED.industry,
    size = EXCLUDED.size,
    stage = EXCLUDED.stage,
    business_model = EXCLUDED.business_model,
    mentor_id = EXCLUDED.mentor_id,
    management_plan = EXCLUDED.management_plan,
    plan_version = organizations.plan_version + 1,
    status = 'draft',
    management_pain_points = EXCLUDED.management_pain_points,
    current_projects = EXCLUDED.current_projects,
    target_framework = EXCLUDED.target_framework,
    team_structure = EXCLUDED.team_structure,
    communication_tools = EXCLUDED.communication_tools,
    culture_preferences = EXCLUDED.culture_preferences,
    updated_at = NOW()
RETURNING *;
