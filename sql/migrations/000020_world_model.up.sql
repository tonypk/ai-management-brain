-- ==========================================
-- Migration 000020: Team World Model
-- Persistent knowledge graph from daily check-ins
-- ==========================================

-- 1. Skills discovered from check-in reports
CREATE TABLE world_model_skills (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  skill_name TEXT NOT NULL,
  proficiency TEXT NOT NULL DEFAULT 'medium',
  source TEXT NOT NULL DEFAULT 'inferred',
  confidence NUMERIC(4,3) NOT NULL DEFAULT 0.500,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  mention_count INT NOT NULL DEFAULT 1,
  UNIQUE(tenant_id, employee_id, skill_name)
);
COMMENT ON TABLE world_model_skills IS 'Skills extracted from daily check-ins with confidence decay';

CREATE INDEX idx_wm_skills_tenant_employee ON world_model_skills(tenant_id, employee_id);
CREATE INDEX idx_wm_skills_tenant_skill ON world_model_skills(tenant_id, skill_name);

-- 2. Collaboration relationships between employees
CREATE TABLE world_model_relationships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  employee_a_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  employee_b_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL DEFAULT 'collaborates',
  context TEXT DEFAULT '',
  strength NUMERIC(4,3) NOT NULL DEFAULT 0.500,
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  interaction_count INT NOT NULL DEFAULT 1,
  UNIQUE(tenant_id, employee_a_id, employee_b_id, relation_type)
);
COMMENT ON TABLE world_model_relationships IS 'Collaboration graph edges extracted from check-ins';
COMMENT ON COLUMN world_model_relationships.relation_type IS 'collaborates (symmetric, a<b enforced in app) | mentors | blocks | depends_on (directed, a→b)';

CREATE INDEX idx_wm_relationships_tenant ON world_model_relationships(tenant_id);
CREATE INDEX idx_wm_relationships_a ON world_model_relationships(tenant_id, employee_a_id);
CREATE INDEX idx_wm_relationships_b ON world_model_relationships(tenant_id, employee_b_id);

-- 3. Blocker patterns per employee
CREATE TABLE world_model_blockers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  description TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ,
  recurrence_count INT NOT NULL DEFAULT 1
);
COMMENT ON TABLE world_model_blockers IS 'Recurring blocker patterns per employee';
COMMENT ON COLUMN world_model_blockers.category IS 'cross_team | tooling | requirements | skills_gap | external';

CREATE INDEX idx_wm_blockers_tenant_status ON world_model_blockers(tenant_id, status);
CREATE INDEX idx_wm_blockers_employee ON world_model_blockers(tenant_id, employee_id);

-- 4. Growth events (immutable event log)
CREATE TABLE world_model_growth_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  description TEXT NOT NULL,
  detected_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE world_model_growth_events IS 'Skill milestones and growth events detected from check-ins';
COMMENT ON COLUMN world_model_growth_events.event_type IS 'new_skill | skill_upgrade | first_solo | mentoring_others';

CREATE INDEX idx_wm_growth_tenant_employee ON world_model_growth_events(tenant_id, employee_id);
CREATE INDEX idx_wm_growth_detected ON world_model_growth_events(tenant_id, detected_at DESC);

-- 5. Team-level insights (periodically refreshed)
CREATE TABLE world_model_insights (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  dimension TEXT NOT NULL,
  insight_text TEXT NOT NULL,
  evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
  confidence NUMERIC(4,3) NOT NULL DEFAULT 0.500,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
);
COMMENT ON TABLE world_model_insights IS 'AI-generated team insights refreshed daily';
COMMENT ON COLUMN world_model_insights.dimension IS 'rhythm | context | risk | opportunity';

CREATE INDEX idx_wm_insights_tenant_dim ON world_model_insights(tenant_id, dimension);
CREATE INDEX idx_wm_insights_expires ON world_model_insights(expires_at);
CREATE INDEX idx_wm_insights_active ON world_model_insights(tenant_id, expires_at)
  WHERE expires_at > now();
