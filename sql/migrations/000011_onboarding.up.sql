-- 1. Drop deprecated wizard_sessions table
DROP TABLE IF EXISTS wizard_sessions;

-- 2. Create onboarding_sessions table
CREATE TABLE onboarding_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'onboarding',
    confirm_step    INT NOT NULL DEFAULT 0,
    collected_data  JSONB NOT NULL DEFAULT '{}',
    proposed_plan   JSONB NOT NULL DEFAULT '{}',
    message_count   INT NOT NULL DEFAULT 0,
    channel_type    VARCHAR(20) NOT NULL DEFAULT 'telegram',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_onboarding_sessions_tenant ON onboarding_sessions(tenant_id);

-- 3. Create org_units table (flexible N-level tree)
CREATE TABLE org_units (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    parent_id       UUID REFERENCES org_units(id),
    name            VARCHAR(200) NOT NULL,
    unit_type       VARCHAR(50) NOT NULL,
    head_role       VARCHAR(200),
    head_employee_id UUID REFERENCES employees(id),
    responsibilities TEXT,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_org_units_tenant ON org_units(tenant_id);
CREATE INDEX idx_org_units_parent ON org_units(parent_id) WHERE parent_id IS NOT NULL;

-- 4. Add org_unit_id to employees
ALTER TABLE employees ADD COLUMN IF NOT EXISTS org_unit_id UUID REFERENCES org_units(id);
CREATE INDEX IF NOT EXISTS idx_employees_org_unit ON employees(org_unit_id) WHERE org_unit_id IS NOT NULL;

-- 5. Extend organizations table (all new fields nullable)
ALTER TABLE organizations ALTER COLUMN industry DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN size DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN stage DROP NOT NULL;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS management_pain_points TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS current_projects JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS target_framework VARCHAR(50);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS team_structure JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS communication_tools TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS culture_preferences JSONB;

-- 6. Extend tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_slack_id TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_lark_id TEXT;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_slack ON tenants(boss_slack_id) WHERE boss_slack_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_lark ON tenants(boss_lark_id) WHERE boss_lark_id IS NOT NULL;
