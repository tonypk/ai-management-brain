-- Goals/OKR tables for Management 101 Planning module
CREATE TABLE goals (
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
CREATE INDEX idx_goals_tenant_cycle ON goals(tenant_id, cycle);
CREATE INDEX idx_goals_owner ON goals(owner_id) WHERE owner_id IS NOT NULL;

CREATE TABLE key_results (
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
CREATE INDEX idx_key_results_goal ON key_results(goal_id);

CREATE TABLE goal_snapshots (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id          UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    overall_progress NUMERIC(5,2) NOT NULL DEFAULT 0,
    snapshot_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_goal_snapshots_goal_date ON goal_snapshots(goal_id, snapshot_date);
CREATE UNIQUE INDEX idx_goal_snapshots_unique ON goal_snapshots(goal_id, snapshot_date);
