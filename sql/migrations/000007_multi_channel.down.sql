-- 000007_multi_channel.down.sql
ALTER TABLE chase_logs DROP COLUMN IF EXISTS channel;
ALTER TABLE reports DROP COLUMN IF EXISTS channel;

ALTER TABLE tenants DROP COLUMN IF EXISTS enabled_channels;
ALTER TABLE tenants DROP COLUMN IF EXISTS signal_phone;
ALTER TABLE tenants DROP COLUMN IF EXISTS lark_app_secret;
ALTER TABLE tenants DROP COLUMN IF EXISTS lark_app_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS slack_signing_secret;
ALTER TABLE tenants DROP COLUMN IF EXISTS slack_bot_token;

DROP INDEX IF EXISTS idx_employees_lark;
DROP INDEX IF EXISTS idx_employees_slack;
DROP INDEX IF EXISTS idx_employees_signal;

ALTER TABLE employees DROP COLUMN IF EXISTS preferred_channel;
ALTER TABLE employees DROP COLUMN IF EXISTS lark_id;
ALTER TABLE employees DROP COLUMN IF EXISTS slack_id;
ALTER TABLE employees DROP COLUMN IF EXISTS signal_phone;
