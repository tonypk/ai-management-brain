-- ==========================================
-- Migration 000016: HalaOS Bridge
-- Employee linking + webhook config + event log
-- ==========================================

-- Employee linking fields
ALTER TABLE employees ADD COLUMN halaos_employee_id BIGINT;
ALTER TABLE employees ADD COLUMN halaos_employee_no TEXT;
CREATE UNIQUE INDEX idx_employees_halaos_id
  ON employees(tenant_id, halaos_employee_id)
  WHERE halaos_employee_id IS NOT NULL;

-- HalaOS webhook configuration per tenant
CREATE TABLE halaos_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  webhook_secret TEXT NOT NULL,
  halaos_company_id BIGINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HalaOS event audit log + idempotency
CREATE TABLE halaos_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  payload JSONB NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_halaos_events_idempotency
  ON halaos_events(tenant_id, idempotency_key);
CREATE INDEX idx_halaos_events_tenant_time
  ON halaos_events(tenant_id, processed_at DESC);
