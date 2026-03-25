CREATE TABLE IF NOT EXISTS wizard_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    mentor_id       TEXT NOT NULL,
    current_step    TEXT NOT NULL DEFAULT 'start',
    conversation    JSONB NOT NULL DEFAULT '[]',
    company_profile JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TABLE IF EXISTS onboarding_sessions;

ALTER TABLE employees DROP COLUMN IF EXISTS org_unit_id;
DROP TABLE IF EXISTS org_units;

ALTER TABLE organizations DROP COLUMN IF EXISTS management_pain_points;
ALTER TABLE organizations DROP COLUMN IF EXISTS current_projects;
ALTER TABLE organizations DROP COLUMN IF EXISTS target_framework;
ALTER TABLE organizations DROP COLUMN IF EXISTS team_structure;
ALTER TABLE organizations DROP COLUMN IF EXISTS communication_tools;
ALTER TABLE organizations DROP COLUMN IF EXISTS culture_preferences;

ALTER TABLE tenants DROP COLUMN IF EXISTS onboarding_completed_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS boss_slack_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS boss_lark_id;
