-- ==========================================
-- Migration 000015: Brain Layer v2
-- Context Layer + State Engine + Incentives
-- ==========================================

-- 1. Extend employees with execution context
ALTER TABLE employees
  ADD COLUMN IF NOT EXISTS execution_score NUMERIC(5,2) DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_load TEXT DEFAULT 'medium',
  ADD COLUMN IF NOT EXISTS strengths JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS risk_flags JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS work_scope JSONB DEFAULT '[]'::jsonb;

-- 2. Extend organizations with strategic context
ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS strategic_priorities JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS key_risks JSONB DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS management_style_weights JSONB DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS countries JSONB DEFAULT '[]'::jsonb;

-- 3. Extend goals with level and source
ALTER TABLE goals
  ADD COLUMN IF NOT EXISTS level TEXT NOT NULL DEFAULT 'team',
  ADD COLUMN IF NOT EXISTS goal_type TEXT NOT NULL DEFAULT 'okr',
  ADD COLUMN IF NOT EXISTS source_system TEXT,
  ADD COLUMN IF NOT EXISTS source_ref TEXT;

-- 4. Extend key_results with metric link and status
ALTER TABLE key_results
  ADD COLUMN IF NOT EXISTS metric_id UUID,
  ADD COLUMN IF NOT EXISTS baseline_value NUMERIC(12,2),
  ADD COLUMN IF NOT EXISTS formula_note TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'on_track',
  ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES employees(id);

-- ==========================================
-- New tables
-- ==========================================

-- 5. metrics — KPI definitions
CREATE TABLE metrics (
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

CREATE INDEX idx_metrics_tenant ON metrics(tenant_id) WHERE is_active = true;
CREATE UNIQUE INDEX idx_metrics_tenant_name ON metrics(tenant_id, name);

-- 6. metric_values — time-series
CREATE TABLE metric_values (
  id BIGSERIAL PRIMARY KEY,
  metric_id UUID NOT NULL REFERENCES metrics(id) ON DELETE CASCADE,
  observed_at TIMESTAMPTZ NOT NULL,
  value NUMERIC(12,4) NOT NULL,
  dimensions JSONB DEFAULT '{}'::jsonb,
  source_ref TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_metric_values_metric_time ON metric_values(metric_id, observed_at DESC);

-- 7. projects
CREATE TABLE projects (
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

CREATE INDEX idx_projects_tenant ON projects(tenant_id);
CREATE INDEX idx_projects_owner ON projects(owner_id) WHERE owner_id IS NOT NULL;

-- 8. tasks
CREATE TABLE tasks (
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

CREATE INDEX idx_tasks_tenant ON tasks(tenant_id);
CREATE INDEX idx_tasks_owner ON tasks(owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX idx_tasks_project ON tasks(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_tasks_status ON tasks(tenant_id, status) WHERE status NOT IN ('done', 'cancelled');

-- 9. reporting_lines
CREATE TABLE reporting_lines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  manager_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  report_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  relationship_type TEXT NOT NULL DEFAULT 'direct',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_reporting_line UNIQUE(manager_id, report_id)
);

CREATE INDEX idx_reporting_lines_manager ON reporting_lines(manager_id);
CREATE INDEX idx_reporting_lines_report ON reporting_lines(report_id);

-- 10. workflows
CREATE TABLE workflows (
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

CREATE INDEX idx_workflows_tenant ON workflows(tenant_id) WHERE is_active = true;

-- 11. incentive_rules
CREATE TABLE incentive_rules (
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

CREATE INDEX idx_incentive_rules_tenant ON incentive_rules(tenant_id) WHERE is_active = true;

-- 12. incentive_scores
CREATE TABLE incentive_scores (
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
  reviewed_at TIMESTAMPTZ,
  CONSTRAINT uq_incentive_score UNIQUE(rule_id, person_id, period)
);

CREATE INDEX idx_incentive_scores_person ON incentive_scores(person_id, period);

-- 13. communication_events
CREATE TABLE communication_events (
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

CREATE INDEX idx_comm_events_tenant_time ON communication_events(tenant_id, occurred_at DESC);
CREATE INDEX idx_comm_events_actor ON communication_events(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_comm_events_type ON communication_events(tenant_id, event_type);

-- 14. execution_signals
CREATE TABLE execution_signals (
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

CREATE INDEX idx_exec_signals_tenant ON execution_signals(tenant_id, generated_at DESC);
CREATE INDEX idx_exec_signals_subject ON execution_signals(subject_type, subject_id, generated_at DESC);

-- 15. working_memory_snapshots
CREATE TABLE working_memory_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  snapshot_type TEXT NOT NULL,
  content JSONB NOT NULL,
  generated_by TEXT DEFAULT 'system',
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wm_snapshots_tenant_type ON working_memory_snapshots(tenant_id, snapshot_type, generated_at DESC);
