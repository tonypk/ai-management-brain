-- ==========================================
-- Migration 000016: HalaOS Bridge (rollback)
-- ==========================================

DROP INDEX IF EXISTS idx_halaos_events_tenant_time;
DROP INDEX IF EXISTS idx_halaos_events_idempotency;
DROP TABLE IF EXISTS halaos_events;
DROP TABLE IF EXISTS halaos_links;
DROP INDEX IF EXISTS idx_employees_halaos_id;
ALTER TABLE employees DROP COLUMN IF EXISTS halaos_employee_no;
ALTER TABLE employees DROP COLUMN IF EXISTS halaos_employee_id;
