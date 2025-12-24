-- Drop trigger
DROP TRIGGER IF EXISTS update_recommendations_updated_at ON recommendations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_recommendations_created_at;
DROP INDEX IF EXISTS idx_recommendations_status;
DROP INDEX IF EXISTS idx_recommendations_workload_type;
DROP INDEX IF EXISTS idx_recommendations_workload;

-- Drop table
DROP TABLE IF EXISTS recommendations;

