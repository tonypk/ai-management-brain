CREATE TABLE IF NOT EXISTS seats (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    seat_type   VARCHAR(50) NOT NULL,
    title       VARCHAR(100) NOT NULL,
    persona_id  VARCHAR(50) NOT NULL,
    scope       TEXT NOT NULL DEFAULT '',
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, seat_type)
);

CREATE INDEX IF NOT EXISTS idx_seats_tenant ON seats(tenant_id);
