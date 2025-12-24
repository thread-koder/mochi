-- Drop indexes
DROP INDEX IF EXISTS idx_history_created_by;
DROP INDEX IF EXISTS idx_history_action;
DROP INDEX IF EXISTS idx_history_created_at;
DROP INDEX IF EXISTS idx_history_entity;

-- Drop table
DROP TABLE IF EXISTS history;

