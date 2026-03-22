-- 000008_employee_profile.up.sql

-- Employee profile fields for AI context
ALTER TABLE employees ADD COLUMN job_title       TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN responsibilities TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN country         TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN language        TEXT NOT NULL DEFAULT '';
