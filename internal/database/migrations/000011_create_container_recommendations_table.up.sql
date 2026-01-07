-- Create container_recommendations table
CREATE TABLE IF NOT EXISTS container_recommendations (
    id BIGSERIAL PRIMARY KEY,
    container_id BIGINT NOT NULL UNIQUE REFERENCES containers(id) ON DELETE CASCADE,
    pod_uid VARCHAR(255) NOT NULL,
    container_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    current_cpu_request VARCHAR(50),
    current_cpu_limit VARCHAR(50),
    current_memory_request VARCHAR(50),
    current_memory_limit VARCHAR(50),
    recommended_cpu_request VARCHAR(50),
    recommended_cpu_limit VARCHAR(50),
    recommended_memory_request VARCHAR(50),
    recommended_memory_limit VARCHAR(50),
    recommendation_mode VARCHAR(50) NOT NULL DEFAULT 'burstable' CHECK (recommendation_mode IN ('burstable', 'guaranteed')),
    confidence_score DECIMAL(5, 4) NOT NULL DEFAULT 0.0 CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMP WITH TIME ZONE
);

-- Create index on container_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_container_recommendations_container_id ON container_recommendations(container_id);

-- Create index on pod_uid and container_name for faster lookups
CREATE INDEX IF NOT EXISTS idx_container_recommendations_pod_uid_container_name ON container_recommendations(pod_uid, container_name);

-- Create index on namespace for faster lookups
CREATE INDEX IF NOT EXISTS idx_container_recommendations_namespace ON container_recommendations(namespace);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_container_recommendations_status ON container_recommendations(status);

-- Create index on created_at for sorting
CREATE INDEX IF NOT EXISTS idx_container_recommendations_created_at ON container_recommendations(created_at DESC);

-- Create index on recommendation_mode for filtering
CREATE INDEX IF NOT EXISTS idx_container_recommendations_recommendation_mode ON container_recommendations(recommendation_mode);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_container_recommendations_updated_at
    BEFORE UPDATE ON container_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
