CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    timezone      TEXT NOT NULL DEFAULT 'Asia/Singapore',
    anthropic_key TEXT,              -- AES-256-GCM encrypted
    mentor_id     TEXT NOT NULL DEFAULT 'inamori',
    mentor_blend  JSONB,             -- optional: {"inamori": 0.7, "dalio": 0.3}
    bot_token     TEXT,              -- AES-256-GCM encrypted
    boss_chat_id  BIGINT NOT NULL,
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    telegram_id   BIGINT UNIQUE,
    culture_code  TEXT NOT NULL DEFAULT 'default',
    role          TEXT NOT NULL DEFAULT 'member',  -- boss | manager | member
    invite_code   TEXT,              -- for /join registration
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    answers       JSONB NOT NULL,    -- {"q1": "...", "q2": "...", "q3": "..."}
    blockers      TEXT,              -- AI-extracted blockers
    sentiment     TEXT,              -- positive | neutral | negative
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(employee_id, report_date)
);

CREATE TABLE chase_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    employee_id   UUID NOT NULL REFERENCES employees(id),
    report_date   DATE NOT NULL,
    step          INT NOT NULL DEFAULT 1,
    action        TEXT NOT NULL,      -- private_message | public_reminder | manager_notify | send_failed
    message       TEXT NOT NULL,
    chased_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE summaries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    summary_date    DATE NOT NULL,
    content         TEXT NOT NULL,
    submission_rate FLOAT NOT NULL DEFAULT 0,
    blockers_count  INT NOT NULL DEFAULT 0,
    key_metrics     JSONB,           -- mentor-specific metrics
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, summary_date)
);

-- Indexes
CREATE INDEX idx_employees_tenant ON employees(tenant_id);
CREATE INDEX idx_employees_telegram ON employees(telegram_id);
CREATE INDEX idx_reports_tenant_date ON reports(tenant_id, report_date);
CREATE INDEX idx_reports_employee_date ON reports(employee_id, report_date);
CREATE INDEX idx_chase_logs_tenant_date ON chase_logs(tenant_id, report_date);
CREATE INDEX idx_chase_logs_employee ON chase_logs(employee_id, report_date);
CREATE INDEX idx_summaries_tenant_date ON summaries(tenant_id, summary_date);
