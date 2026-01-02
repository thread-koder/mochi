-- Drop table
DROP TABLE IF EXISTS recommendations;

-- Drop function
-- Note: This function is shared across multiple tables
DROP FUNCTION IF EXISTS update_updated_at_column();
