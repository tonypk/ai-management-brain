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
