-- Reverse migration 000015

DROP TABLE IF EXISTS working_memory_snapshots;
DROP TABLE IF EXISTS execution_signals;
DROP TABLE IF EXISTS communication_events;
DROP TABLE IF EXISTS incentive_scores;
DROP TABLE IF EXISTS incentive_rules;
DROP TABLE IF EXISTS workflows;
DROP TABLE IF EXISTS reporting_lines;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS metric_values;
DROP TABLE IF EXISTS metrics;

-- Remove extended columns from key_results
ALTER TABLE key_results
  DROP COLUMN IF EXISTS metric_id,
  DROP COLUMN IF EXISTS baseline_value,
  DROP COLUMN IF EXISTS formula_note,
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS owner_id;

-- Remove extended columns from goals
ALTER TABLE goals
  DROP COLUMN IF EXISTS level,
  DROP COLUMN IF EXISTS goal_type,
  DROP COLUMN IF EXISTS source_system,
  DROP COLUMN IF EXISTS source_ref;

-- Remove extended columns from organizations
ALTER TABLE organizations
  DROP COLUMN IF EXISTS strategic_priorities,
  DROP COLUMN IF EXISTS key_risks,
  DROP COLUMN IF EXISTS management_style_weights,
  DROP COLUMN IF EXISTS countries;

-- Remove extended columns from employees
ALTER TABLE employees
  DROP COLUMN IF EXISTS execution_score,
  DROP COLUMN IF EXISTS current_load,
  DROP COLUMN IF EXISTS strengths,
  DROP COLUMN IF EXISTS risk_flags,
  DROP COLUMN IF EXISTS work_scope;
