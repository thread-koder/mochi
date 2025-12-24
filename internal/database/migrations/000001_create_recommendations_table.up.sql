-- Create recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id BIGSERIAL PRIMARY KEY,
    workload_type VARCHAR(50) NOT NULL CHECK (workload_type IN ('deployment', 'statefulset', 'daemonset', 'job', 'cronjob', 'pod')),
    workload_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    current_value VARCHAR(255) NOT NULL,
    recommended_value VARCHAR(255) NOT NULL,
    confidence DECIMAL(5, 4) NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
    reason TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMP WITH TIME ZONE
);

-- Create index on namespace, workload_type, and workload_name for faster lookups
CREATE INDEX IF NOT EXISTS idx_recommendations_workload ON recommendations(namespace, workload_type, workload_name);

-- Create index on workload_type for filtering
CREATE INDEX IF NOT EXISTS idx_recommendations_workload_type ON recommendations(workload_type);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_recommendations_status ON recommendations(status);

-- Create index on created_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_recommendations_created_at ON recommendations(created_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_recommendations_updated_at
    BEFORE UPDATE ON recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

