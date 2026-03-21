CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memories (
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

CREATE INDEX idx_memories_tenant_type ON memories(tenant_id, memory_type, memory_tier);
CREATE INDEX idx_memories_employee ON memories(employee_id) WHERE employee_id IS NOT NULL;
CREATE INDEX idx_memories_expires ON memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_memories_merged ON memories(merged_into) WHERE merged_into IS NOT NULL;
