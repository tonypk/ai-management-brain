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

CREATE INDEX idx_recommendations_tenant_status ON recommendations(tenant_id, status);
CREATE INDEX idx_recommendations_tenant_created ON recommendations(tenant_id, created_at DESC);
CREATE INDEX idx_recommendations_expires ON recommendations(tenant_id, expires_at) WHERE status = 'pending';
