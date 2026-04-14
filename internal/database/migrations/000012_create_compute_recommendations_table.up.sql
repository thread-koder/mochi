CREATE TABLE IF NOT EXISTS compute_recommendations (
    id BIGSERIAL PRIMARY KEY,
    workload_type VARCHAR(50) NOT NULL,
    workload_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    recommendation_mode VARCHAR(50) NOT NULL DEFAULT 'burstable'
        CHECK (recommendation_mode IN ('cost_optimized', 'burstable', 'guaranteed')),
    recommendations JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'rejected', 'superseded')),
    analysis_time_range INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compute_recommendations_workload 
    ON compute_recommendations(namespace, workload_type, workload_name, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_compute_recommendations_namespace 
    ON compute_recommendations(namespace, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_compute_recommendations_status 
    ON compute_recommendations(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_compute_recommendations_mode 
    ON compute_recommendations(recommendation_mode);

CREATE TRIGGER update_compute_recommendations_updated_at
    BEFORE UPDATE ON compute_recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
