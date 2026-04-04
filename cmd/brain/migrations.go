package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations applies database migrations idempotently.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationSQL := `
CREATE TABLE IF NOT EXISTS tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,
    bot_token     TEXT,
    boss_chat_id  BIGINT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    telegram_id   BIGINT UNIQUE,
    culture_code  TEXT NOT NULL DEFAULT 'default',
    role          TEXT NOT NULL DEFAULT 'member',
    invite_code   TEXT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,
    blockers      TEXT,
    sentiment     TEXT,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);
CREATE TABLE IF NOT EXISTS chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,
    message       TEXT NOT NULL,
    chased_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary_date    DATE NOT NULL,
    content         TEXT NOT NULL,
    submission_rate FLOAT NOT NULL DEFAULT 0,
    blockers_count  INT NOT NULL DEFAULT 0,
    key_metrics     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);
CREATE INDEX IF NOT EXISTS idx_employees_tenant ON employees(tenant_id);
CREATE INDEX IF NOT EXISTS idx_employees_telegram ON employees(telegram_id);
CREATE INDEX IF NOT EXISTS idx_reports_tenant_date ON reports(tenant_id, report_date);
CREATE INDEX IF NOT EXISTS idx_reports_employee_date ON reports(employee_id, report_date);
CREATE INDEX IF NOT EXISTS idx_chase_logs_tenant_date ON chase_logs(tenant_id, report_date);
CREATE INDEX IF NOT EXISTS idx_chase_logs_employee ON chase_logs(employee_id, report_date);
CREATE INDEX IF NOT EXISTS idx_summaries_tenant_date ON summaries(tenant_id, summary_date);
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'boss',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
`
	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return err
	}

	// Migration 000002: API Keys + tenant plan column
	migration002 := `
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan TEXT NOT NULL DEFAULT 'free';
CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    prefix       TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT 'default',
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    is_active    BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id) WHERE is_active = true;
`
	if _, err := pool.Exec(ctx, migration002); err != nil {
		return err
	}

	// Migration 000003: Organizations + Wizard Sessions
	migration003 := `
CREATE TABLE IF NOT EXISTS organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    industry        TEXT NOT NULL,
    size            INT NOT NULL,
    stage           TEXT NOT NULL,
    business_model  TEXT,
    region          TEXT,
    mentor_id       TEXT NOT NULL,
    management_plan JSONB NOT NULL DEFAULT '{}',
    plan_version    INT NOT NULL DEFAULT 1,
    status          TEXT NOT NULL DEFAULT 'draft',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
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
CREATE INDEX IF NOT EXISTS idx_organizations_tenant ON organizations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wizard_sessions_tenant ON wizard_sessions(tenant_id);
`
	if _, err := pool.Exec(ctx, migration003); err != nil {
		return err
	}

	// Migration 000004: AI Role Instances + Suggestions
	migration004 := `
CREATE TABLE IF NOT EXISTS ai_role_instances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role_id     TEXT NOT NULL,
    title       TEXT NOT NULL,
    mentor_id   TEXT NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, role_id)
);
CREATE TABLE IF NOT EXISTS ai_suggestions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    role_id      TEXT NOT NULL,
    role_title   TEXT NOT NULL,
    capability   TEXT NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    context_data JSONB NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'pending',
    reviewed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ai_role_instances_tenant ON ai_role_instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_suggestions_tenant_status ON ai_suggestions(tenant_id, status);
`
	if _, err := pool.Exec(ctx, migration004); err != nil {
		return err
	}

	// Migration 000005: Memories table with pgvector
	migration005 := `
CREATE EXTENSION IF NOT EXISTS vector;
CREATE TABLE IF NOT EXISTS memories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    memory_type  VARCHAR(30) NOT NULL,
    memory_tier  VARCHAR(20) NOT NULL DEFAULT 'short_term',
    employee_id  UUID REFERENCES employees(id),
    source_type  VARCHAR(30),
    source_id    UUID,
    content      TEXT NOT NULL,
    summary      TEXT,
    embedding    vector(384),
    importance   FLOAT DEFAULT 0.5,
    access_count INT DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    merged_into  UUID REFERENCES memories(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX IF NOT EXISTS idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
`
	if _, err := pool.Exec(ctx, migration005); err != nil {
		return err
	}

	// Migration 000006: see sql/migrations/000006_vector384.up.sql
	migration006 := `ALTER TABLE memories ALTER COLUMN embedding TYPE vector(384);`
	if _, err := pool.Exec(ctx, migration006); err != nil {
		return err
	}

	// Migration 000007: Multi-channel support
	migration007 := `
ALTER TABLE employees ADD COLUMN IF NOT EXISTS signal_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS slack_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS lark_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN IF NOT EXISTS preferred_channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_signal ON employees(signal_phone) WHERE signal_phone IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_slack ON employees(slack_id) WHERE slack_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_lark ON employees(lark_id) WHERE lark_id IS NOT NULL;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS slack_bot_token TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS slack_signing_secret TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS lark_app_id TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS lark_app_secret TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS signal_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS enabled_channels TEXT[] NOT NULL DEFAULT '{telegram}';
ALTER TABLE reports ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
ALTER TABLE chase_logs ADD COLUMN IF NOT EXISTS channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
`
	if _, err := pool.Exec(ctx, migration007); err != nil {
		return err
	}

	const migration008 = `
-- 000008: employee profile fields
ALTER TABLE employees ADD COLUMN IF NOT EXISTS job_title       TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS country         TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN IF NOT EXISTS language        TEXT NOT NULL DEFAULT '';
`
	if _, err := pool.Exec(ctx, migration008); err != nil {
		return err
	}

	const migration009 = `
-- 000009: group chats
CREATE TABLE IF NOT EXISTS group_chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    platform VARCHAR(20) NOT NULL DEFAULT 'telegram',
    platform_chat_id VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    group_type VARCHAR(50) NOT NULL DEFAULT 'general',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(platform, platform_chat_id)
);
CREATE INDEX IF NOT EXISTS idx_group_chats_tenant ON group_chats(tenant_id) WHERE is_active = true;
`
	if _, err := pool.Exec(ctx, migration009); err != nil {
		return err
	}

	const migration010 = `
-- 000010: C-Suite seats
CREATE TABLE IF NOT EXISTS seats (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    seat_type   VARCHAR(50) NOT NULL,
    title       VARCHAR(100) NOT NULL,
    persona_id  VARCHAR(50) NOT NULL,
    scope       TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, seat_type)
);
CREATE INDEX IF NOT EXISTS idx_seats_tenant ON seats(tenant_id);
`
	if _, err := pool.Exec(ctx, migration010); err != nil {
		return err
	}

	const migration011 = `
-- 000011: Onboarding + Org Units
DROP TABLE IF EXISTS wizard_sessions;
CREATE TABLE IF NOT EXISTS onboarding_sessions (
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_onboarding_sessions_tenant ON onboarding_sessions(tenant_id);
CREATE TABLE IF NOT EXISTS org_units (
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
CREATE INDEX IF NOT EXISTS idx_org_units_tenant ON org_units(tenant_id);
CREATE INDEX IF NOT EXISTS idx_org_units_parent ON org_units(parent_id) WHERE parent_id IS NOT NULL;
ALTER TABLE employees ADD COLUMN IF NOT EXISTS org_unit_id UUID REFERENCES org_units(id);
CREATE INDEX IF NOT EXISTS idx_employees_org_unit ON employees(org_unit_id) WHERE org_unit_id IS NOT NULL;
ALTER TABLE organizations ALTER COLUMN industry DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN size DROP NOT NULL;
ALTER TABLE organizations ALTER COLUMN stage DROP NOT NULL;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS management_pain_points TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS current_projects JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS target_framework VARCHAR(50);
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS team_structure JSONB;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS communication_tools TEXT[];
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS culture_preferences JSONB;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_slack_id TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS boss_lark_id TEXT;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_slack ON tenants(boss_slack_id) WHERE boss_slack_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_boss_lark ON tenants(boss_lark_id) WHERE boss_lark_id IS NOT NULL;
`
	_, err := pool.Exec(ctx, migration011)
	if err != nil {
		return err
	}

	// Migration 000012: Goals / OKR tables
	const migration012 = `
CREATE TABLE IF NOT EXISTS goals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    owner_id    UUID REFERENCES employees(id),
    title       VARCHAR(500) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'active', 'completed', 'cancelled')),
    cycle       VARCHAR(10) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_goals_tenant_cycle ON goals(tenant_id, cycle);
CREATE INDEX IF NOT EXISTS idx_goals_owner ON goals(owner_id) WHERE owner_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS key_results (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id       UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title         VARCHAR(500) NOT NULL,
    target        NUMERIC(12,2) NOT NULL DEFAULT 0,
    current_value NUMERIC(12,2) NOT NULL DEFAULT 0,
    unit          VARCHAR(20) NOT NULL DEFAULT '%',
    due_date      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_key_results_goal ON key_results(goal_id);

CREATE TABLE IF NOT EXISTS goal_snapshots (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id          UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    overall_progress NUMERIC(5,2) NOT NULL DEFAULT 0,
    snapshot_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_goal_snapshots_goal_date ON goal_snapshots(goal_id, snapshot_date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_goal_snapshots_unique ON goal_snapshots(goal_id, snapshot_date);
`
	if _, err := pool.Exec(ctx, migration012); err != nil {
		return err
	}

	// Migration 000013: Reviews, Meetings, Skills
	const migration013 = `
CREATE TABLE IF NOT EXISTS review_cycles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    title       VARCHAR(200) NOT NULL,
    period      VARCHAR(20) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'active', 'completed')),
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_review_cycles_tenant ON review_cycles(tenant_id, status);

CREATE TABLE IF NOT EXISTS performance_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id        UUID NOT NULL REFERENCES review_cycles(id) ON DELETE CASCADE,
    employee_id     UUID NOT NULL REFERENCES employees(id),
    reviewer_id     UUID REFERENCES employees(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'in_progress', 'submitted', 'acknowledged')),
    self_rating     SMALLINT CHECK (self_rating BETWEEN 1 AND 5),
    manager_rating  SMALLINT CHECK (manager_rating BETWEEN 1 AND 5),
    self_summary    TEXT NOT NULL DEFAULT '',
    manager_summary TEXT NOT NULL DEFAULT '',
    strengths       TEXT NOT NULL DEFAULT '',
    improvements    TEXT NOT NULL DEFAULT '',
    submitted_at    TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_reviews_cycle ON performance_reviews(cycle_id);
CREATE INDEX IF NOT EXISTS idx_reviews_employee ON performance_reviews(employee_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_unique ON performance_reviews(cycle_id, employee_id);

CREATE TABLE IF NOT EXISTS meetings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    employee_id    UUID NOT NULL REFERENCES employees(id),
    manager_id     UUID REFERENCES employees(id),
    meeting_date   DATE NOT NULL,
    duration_min   SMALLINT NOT NULL DEFAULT 30,
    notes          TEXT NOT NULL DEFAULT '',
    mood           VARCHAR(20) NOT NULL DEFAULT ''
                   CHECK (mood IN ('', 'great', 'good', 'neutral', 'concerning', 'critical')),
    follow_up      TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_meetings_tenant ON meetings(tenant_id, meeting_date DESC);
CREATE INDEX IF NOT EXISTS idx_meetings_employee ON meetings(employee_id, meeting_date DESC);

CREATE TABLE IF NOT EXISTS meeting_action_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id  UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    title       VARCHAR(500) NOT NULL,
    assignee_id UUID REFERENCES employees(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'open'
                CHECK (status IN ('open', 'in_progress', 'done')),
    due_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_action_items_meeting ON meeting_action_items(meeting_id);
CREATE INDEX IF NOT EXISTS idx_action_items_assignee ON meeting_action_items(assignee_id) WHERE assignee_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(200) NOT NULL,
    category    VARCHAR(100) NOT NULL DEFAULT 'general',
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_skills_unique ON skills(tenant_id, name);

CREATE TABLE IF NOT EXISTS employee_skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id    UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    level       SMALLINT NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 5),
    notes       TEXT NOT NULL DEFAULT '',
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_employee_skills_unique ON employee_skills(employee_id, skill_id);
CREATE INDEX IF NOT EXISTS idx_employee_skills_skill ON employee_skills(skill_id);
`
	if _, err := pool.Exec(ctx, migration013); err != nil {
		return err
	}

	// Migration 000014: Training + Career
	const migration014 = `
CREATE TABLE IF NOT EXISTS training_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    duration_hours INT NOT NULL DEFAULT 0,
    provider TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    is_mandatory BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_training_programs_tenant ON training_programs(tenant_id);

CREATE TABLE IF NOT EXISTS training_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES training_programs(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'enrolled' CHECK (status IN ('enrolled', 'in_progress', 'completed', 'dropped')),
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    score INT,
    notes TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_training_enrollments_unique ON training_enrollments(program_id, employee_id);
CREATE INDEX IF NOT EXISTS idx_training_enrollments_program ON training_enrollments(program_id);
CREATE INDEX IF NOT EXISTS idx_training_enrollments_employee ON training_enrollments(employee_id);

CREATE TABLE IF NOT EXISTS career_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title TEXT NOT NULL,
    level_order INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    requirements TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_career_levels_tenant ON career_levels(tenant_id);

CREATE TABLE IF NOT EXISTS career_paths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    current_level_id UUID REFERENCES career_levels(id),
    target_level_id UUID REFERENCES career_levels(id),
    target_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_career_paths_unique ON career_paths(employee_id);
CREATE INDEX IF NOT EXISTS idx_career_paths_employee ON career_paths(employee_id);
`
	if _, err := pool.Exec(ctx, migration014); err != nil {
		return err
	}

	// Migration 000015: Brain Layer v2 — Context + State + Incentives
	const migration015 = `
-- Extend employees with execution context
ALTER TABLE employees
  ADD COLUMN IF NOT EXISTS execution_score NUMERIC(5,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_load TEXT DEFAULT 'medium',
  ADD COLUMN IF NOT EXISTS strengths JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS risk_flags JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS work_scope JSONB DEFAULT '[]'::jsonb;

-- Extend organizations with strategic context
ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS strategic_priorities JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS key_risks JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS management_style_weights JSONB DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS countries JSONB DEFAULT '[]'::jsonb;

-- Extend goals with level and source
ALTER TABLE goals
  ADD COLUMN IF NOT EXISTS level TEXT NOT NULL DEFAULT 'team',
  ADD COLUMN IF NOT EXISTS goal_type TEXT NOT NULL DEFAULT 'okr',
  ADD COLUMN IF NOT EXISTS source_system TEXT,
  ADD COLUMN IF NOT EXISTS source_ref TEXT;

-- Extend key_results with metric link and status
ALTER TABLE key_results
  ADD COLUMN IF NOT EXISTS metric_id UUID,
  ADD COLUMN IF NOT EXISTS baseline_value NUMERIC(12,2),
  ADD COLUMN IF NOT EXISTS formula_note TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'on_track',
  ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES employees(id);

-- metrics (KPI definitions)
CREATE TABLE IF NOT EXISTS metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  display_name TEXT NOT NULL,
  formula TEXT NOT NULL DEFAULT '',
  unit TEXT DEFAULT '%',
  source TEXT NOT NULL DEFAULT 'manual',
  refresh_frequency TEXT DEFAULT 'daily',
  target_value NUMERIC(12,2),
  alert_threshold NUMERIC(12,2),
  owner_id UUID REFERENCES employees(id),
  owner_team_id UUID REFERENCES org_units(id),
  tags JSONB DEFAULT '[]'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_metrics_tenant ON metrics(tenant_id) WHERE is_active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_tenant_name ON metrics(tenant_id, name);

-- metric_values (time-series)
CREATE TABLE IF NOT EXISTS metric_values (
  id BIGSERIAL PRIMARY KEY,
  metric_id UUID NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
  observed_at TIMESTAMPTZ NOT NULL,
  value NUMERIC(12,4) NOT NULL,
  dimensions JSONB DEFAULT '{}'::jsonb,
  source_ref TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_metric_values_metric_time ON metric_values(metric_id, observed_at DESC);

-- projects
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  owner_id UUID REFERENCES employees(id),
  owner_team_id UUID REFERENCES org_units(id),
  status TEXT NOT NULL DEFAULT 'planned',
  priority TEXT NOT NULL DEFAULT 'medium',
  linked_goal_ids JSONB DEFAULT '[]'::jsonb,
  linked_metric_ids JSONB DEFAULT '[]'::jsonb,
  blockers JSONB DEFAULT '[]'::jsonb,
  source_system TEXT,
  source_ref TEXT,
  start_date DATE,
  due_date DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_projects_tenant ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id) WHERE owner_id IS NOT NULL;

-- tasks
CREATE TABLE IF NOT EXISTS tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
  goal_id UUID REFERENCES goals(id) ON DELETE SET NULL,
  key_result_id UUID REFERENCES key_results(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  owner_id UUID REFERENCES employees(id),
  owner_team_id UUID REFERENCES org_units(id),
  status TEXT NOT NULL DEFAULT 'todo',
  priority TEXT NOT NULL DEFAULT 'medium',
  due_at TIMESTAMPTZ,
  source_system TEXT DEFAULT 'manual',
  source_ref TEXT,
  created_by_agent BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tasks_tenant ON tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tasks_owner ON tasks(owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(tenant_id, status) WHERE status NOT IN ('done', 'cancelled');

-- reporting_lines
CREATE TABLE IF NOT EXISTS reporting_lines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  manager_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  report_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  relationship_type TEXT NOT NULL DEFAULT 'direct',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_reporting_lines_manager ON reporting_lines(manager_id);
CREATE INDEX IF NOT EXISTS idx_reporting_lines_report ON reporting_lines(report_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_reporting_line') THEN
    ALTER TABLE reporting_lines ADD CONSTRAINT uq_reporting_line UNIQUE(manager_id, report_id);
  END IF;
END $$;

-- workflows
CREATE TABLE IF NOT EXISTS workflows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  category TEXT DEFAULT 'general',
  trigger_conditions JSONB DEFAULT '{}'::jsonb,
  steps JSONB NOT NULL DEFAULT '[]'::jsonb,
  approval_rules JSONB DEFAULT '{}'::jsonb,
  escalation_rules JSONB DEFAULT '{}'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_workflows_tenant ON workflows(tenant_id) WHERE is_active = true;

-- incentive_rules
CREATE TABLE IF NOT EXISTS incentive_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  reward_model TEXT NOT NULL DEFAULT 'individual',
  payout_cycle TEXT NOT NULL DEFAULT 'monthly',
  attribution_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
  penalty_rules JSONB DEFAULT '[]'::jsonb,
  scoring_formula JSONB NOT NULL DEFAULT '{}'::jsonb,
  applies_to JSONB DEFAULT '[]'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_incentive_rules_tenant ON incentive_rules(tenant_id) WHERE is_active = true;

-- incentive_scores
CREATE TABLE IF NOT EXISTS incentive_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  rule_id UUID NOT NULL REFERENCES incentive_rules(id) ON DELETE CASCADE,
  person_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  period TEXT NOT NULL,
  score NUMERIC(8,2) NOT NULL DEFAULT 0,
  score_breakdown JSONB NOT NULL DEFAULT '{}'::jsonb,
  payout_weight NUMERIC(4,3) DEFAULT 1.0,
  attribution_confidence NUMERIC(4,3) DEFAULT 0.8,
  status TEXT NOT NULL DEFAULT 'calculated',
  calculated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  reviewed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_incentive_scores_person ON incentive_scores(person_id, period);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_incentive_score') THEN
    ALTER TABLE incentive_scores ADD CONSTRAINT uq_incentive_score UNIQUE(rule_id, person_id, period);
  END IF;
END $$;

-- communication_events
CREATE TABLE IF NOT EXISTS communication_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL,
  source_id UUID,
  platform TEXT NOT NULL DEFAULT 'telegram',
  event_type TEXT NOT NULL,
  actor_id UUID REFERENCES employees(id),
  target_id UUID REFERENCES employees(id),
  related_task_id UUID REFERENCES tasks(id),
  related_project_id UUID REFERENCES projects(id),
  related_goal_id UUID REFERENCES goals(id),
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  confidence NUMERIC(4,3) DEFAULT 0.8,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_comm_events_tenant_time ON communication_events(tenant_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_comm_events_actor ON communication_events(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_comm_events_type ON communication_events(tenant_id, event_type);

-- execution_signals
CREATE TABLE IF NOT EXISTS execution_signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subject_type TEXT NOT NULL,
  subject_id UUID NOT NULL,
  signal_type TEXT NOT NULL,
  score NUMERIC(5,2) NOT NULL,
  reasons JSONB DEFAULT '[]'::jsonb,
  time_window TEXT DEFAULT '7d',
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_exec_signals_tenant ON execution_signals(tenant_id, generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_exec_signals_subject ON execution_signals(subject_type, subject_id, generated_at DESC);

-- working_memory_snapshots
CREATE TABLE IF NOT EXISTS working_memory_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  snapshot_type TEXT NOT NULL,
  content JSONB NOT NULL,
  generated_by TEXT DEFAULT 'system',
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wm_snapshots_tenant_type ON working_memory_snapshots(tenant_id, snapshot_type, generated_at DESC);
`
	if _, err = pool.Exec(ctx, migration015); err != nil {
		return err
	}

	// Migration 000017: Recommendations
	const migration017 = `
CREATE TABLE IF NOT EXISTS recommendations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id),
    category           TEXT NOT NULL CHECK (category IN ('people', 'project', 'kpi', 'organization')),
    priority           TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    title              TEXT NOT NULL,
    description        TEXT NOT NULL,
    suggested_actions  JSONB NOT NULL DEFAULT '[]',
    evidence           JSONB NOT NULL DEFAULT '{}',
    source             TEXT NOT NULL CHECK (source IN ('daily_scan', 'realtime_trigger')),
    status             TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'dismissed', 'executed', 'expired')),
    target_entity_type TEXT CHECK (target_entity_type IN ('employee', 'project', 'metric', 'goal') OR target_entity_type IS NULL),
    target_entity_id   UUID,
    expires_at         TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at        TIMESTAMPTZ,
    executed_at        TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_recommendations_tenant_status ON recommendations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_recommendations_tenant_created ON recommendations(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_expires ON recommendations(tenant_id, expires_at) WHERE status = 'pending';
`
	_, err = pool.Exec(ctx, migration017)
	return err
}
