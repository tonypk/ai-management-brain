-- Sync configuration (one row per tenant per storage type)
CREATE TABLE sync_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  storage_type TEXT NOT NULL,
  is_enabled BOOLEAN NOT NULL DEFAULT false,
  entity_types TEXT[] NOT NULL DEFAULT '{}',
  sync_frequency_minutes INT NOT NULL DEFAULT 30,
  last_sync_at TIMESTAMPTZ,
  last_sync_status TEXT,
  config JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, storage_type)
);

-- Sync logs
CREATE TABLE sync_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  sync_config_id UUID NOT NULL REFERENCES sync_configs(id),
  direction TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'running',
  items_pushed INT NOT NULL DEFAULT 0,
  items_pulled INT NOT NULL DEFAULT 0,
  conflicts INT NOT NULL DEFAULT 0,
  errors JSONB NOT NULL DEFAULT '[]',
  summary TEXT
);

CREATE INDEX idx_sync_configs_tenant ON sync_configs(tenant_id);
CREATE INDEX idx_sync_logs_config ON sync_logs(sync_config_id);
CREATE INDEX idx_sync_logs_started ON sync_logs(started_at DESC);

-- Add external reference columns to core tables for sync tracking
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE goals ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS external_url TEXT;

ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS external_url TEXT;
