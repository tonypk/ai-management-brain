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
