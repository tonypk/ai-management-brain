-- 000007_multi_channel.up.sql

-- Employee channel columns
ALTER TABLE employees ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE employees ADD COLUMN slack_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN lark_id VARCHAR(50);
ALTER TABLE employees ADD COLUMN preferred_channel VARCHAR(20) NOT NULL DEFAULT 'telegram';

CREATE UNIQUE INDEX idx_employees_signal ON employees(signal_phone) WHERE signal_phone IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_slack ON employees(slack_id) WHERE slack_id IS NOT NULL;
CREATE UNIQUE INDEX idx_employees_lark ON employees(lark_id) WHERE lark_id IS NOT NULL;

-- Tenant channel configuration
ALTER TABLE tenants ADD COLUMN slack_bot_token TEXT;
ALTER TABLE tenants ADD COLUMN slack_signing_secret TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_id TEXT;
ALTER TABLE tenants ADD COLUMN lark_app_secret TEXT;
ALTER TABLE tenants ADD COLUMN signal_phone VARCHAR(20);
ALTER TABLE tenants ADD COLUMN enabled_channels TEXT[] NOT NULL DEFAULT '{telegram}';

-- Track which channel reports/chases came from
ALTER TABLE reports ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
ALTER TABLE chase_logs ADD COLUMN channel VARCHAR(20) NOT NULL DEFAULT 'telegram';
