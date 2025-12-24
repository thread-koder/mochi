-- Drop trigger
DROP TRIGGER IF EXISTS update_costs_updated_at ON costs;

-- Drop indexes
DROP INDEX IF EXISTS idx_costs_type;
DROP INDEX IF EXISTS idx_costs_period;
DROP INDEX IF EXISTS idx_costs_service;
DROP INDEX IF EXISTS idx_costs_team;
DROP INDEX IF EXISTS idx_costs_namespace;

-- Drop table
DROP TABLE IF EXISTS costs;

