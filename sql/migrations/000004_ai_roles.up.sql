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
