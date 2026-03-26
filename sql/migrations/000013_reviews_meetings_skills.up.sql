-- Batch 2: Performance Reviews, 1:1 Meetings, Skill Inventory

-- Performance Reviews
CREATE TABLE review_cycles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    title       VARCHAR(200) NOT NULL,
    period      VARCHAR(20) NOT NULL,  -- "2026-Q1", "2026-H1", "2026"
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'active', 'completed')),
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_review_cycles_tenant ON review_cycles(tenant_id, status);

CREATE TABLE performance_reviews (
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
CREATE INDEX idx_reviews_cycle ON performance_reviews(cycle_id);
CREATE INDEX idx_reviews_employee ON performance_reviews(employee_id);
CREATE UNIQUE INDEX idx_reviews_unique ON performance_reviews(cycle_id, employee_id);

-- 1:1 Meetings
CREATE TABLE meetings (
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
CREATE INDEX idx_meetings_tenant ON meetings(tenant_id, meeting_date DESC);
CREATE INDEX idx_meetings_employee ON meetings(employee_id, meeting_date DESC);

CREATE TABLE meeting_action_items (
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
CREATE INDEX idx_action_items_meeting ON meeting_action_items(meeting_id);
CREATE INDEX idx_action_items_assignee ON meeting_action_items(assignee_id) WHERE assignee_id IS NOT NULL;

-- Skill Inventory
CREATE TABLE skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        VARCHAR(200) NOT NULL,
    category    VARCHAR(100) NOT NULL DEFAULT 'general',
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_skills_unique ON skills(tenant_id, name);

CREATE TABLE employee_skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    skill_id    UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    level       SMALLINT NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 5),
    notes       TEXT NOT NULL DEFAULT '',
    assessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_employee_skills_unique ON employee_skills(employee_id, skill_id);
CREATE INDEX idx_employee_skills_skill ON employee_skills(skill_id);
