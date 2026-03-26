-- ==========================================
-- Migration 000019: Consulting Engagements
-- AI McKinsey-style consulting engine
-- ==========================================

-- 1. engagements — consulting projects
CREATE TABLE engagements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  problem_statement TEXT NOT NULL,
  tier TEXT NOT NULL DEFAULT 'standard',
  category TEXT DEFAULT 'general',
  phase TEXT NOT NULL DEFAULT 'intake',
  diagnosis_questions JSONB DEFAULT '[]'::jsonb,
  diagnosis_answers JSONB DEFAULT '[]'::jsonb,
  diagnosis_data JSONB DEFAULT '{}'::jsonb,
  analysis JSONB DEFAULT '{}'::jsonb,
  plan JSONB DEFAULT '{}'::jsonb,
  progress_pct NUMERIC(5,2) DEFAULT 0,
  next_check_at TIMESTAMPTZ,
  mentor_id TEXT DEFAULT '',
  culture_code TEXT DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at TIMESTAMPTZ
);

CREATE INDEX idx_engagements_tenant_phase ON engagements(tenant_id, phase);
CREATE INDEX idx_engagements_next_check ON engagements(next_check_at)
  WHERE phase IN ('executing', 'tracking');

-- 2. engagement_actions — individual actions within an engagement
CREATE TABLE engagement_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  engagement_id UUID NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
  action_type TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  params JSONB NOT NULL DEFAULT '{}'::jsonb,
  owner_name TEXT DEFAULT '',
  priority TEXT DEFAULT 'medium',
  due_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'pending',
  approved_at TIMESTAMPTZ,
  executed_at TIMESTAMPTZ,
  result JSONB DEFAULT '{}'::jsonb,
  linked_task_id UUID REFERENCES tasks(id),
  linked_meeting_id UUID REFERENCES meetings(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_engagement_actions_engagement ON engagement_actions(engagement_id);
CREATE INDEX idx_engagement_actions_status ON engagement_actions(engagement_id, status);
