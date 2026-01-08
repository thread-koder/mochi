-- Create compute_recommendations table
CREATE TABLE IF NOT EXISTS compute_recommendations (
    id BIGSERIAL PRIMARY KEY,
    workload_type VARCHAR(50) NOT NULL,
    workload_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    recommendation_mode VARCHAR(50) NOT NULL DEFAULT 'burstable'
        CHECK (recommendation_mode IN ('burstable', 'guaranteed')),
    recommendations JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'rejected', 'superseded')),
    analysis_time_range INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on namespace, workload_type, workload_name and created_at for faster lookups
CREATE INDEX IF NOT EXISTS idx_compute_recommendations_workload 
    ON compute_recommendations(namespace, workload_type, workload_name, created_at DESC);

-- Create index on namespace and status for faster lookups
CREATE INDEX IF NOT EXISTS idx_compute_recommendations_namespace 
    ON compute_recommendations(namespace, status, created_at DESC);

-- Create index on status for faster lookups
CREATE INDEX IF NOT EXISTS idx_compute_recommendations_status 
    ON compute_recommendations(status, created_at DESC);

-- Create index on recommendation_mode for faster lookups
CREATE INDEX IF NOT EXISTS idx_compute_recommendations_mode 
    ON compute_recommendations(recommendation_mode);

-- Create index on id for unique lookups
CREATE INDEX IF NOT EXISTS idx_compute_recommendations_id 
    ON compute_recommendations(id);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_compute_recommendations_updated_at
    BEFORE UPDATE ON compute_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
