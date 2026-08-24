CREATE TABLE IF NOT EXISTS pod_attributions (
    uid VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    workload_kind VARCHAR(50),
    workload_name VARCHAR(255),
    phase VARCHAR(50) NOT NULL,
    node VARCHAR(255),
    containers JSONB NOT NULL DEFAULT '[]',
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_pod_attributions_namespace_workload_last_seen
    ON pod_attributions(namespace, workload_kind, workload_name, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_pod_attributions_last_seen
    ON pod_attributions(last_seen_at);
