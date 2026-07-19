CREATE TABLE IF NOT EXISTS containers (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    pod_uid VARCHAR(255) NOT NULL REFERENCES pods(uid) ON DELETE CASCADE,
    pod_name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    image VARCHAR(512) NOT NULL,
    image_pull_policy VARCHAR(50) NOT NULL,
    ports JSONB,
    cpu_request VARCHAR(50),
    cpu_limit VARCHAR(50),
    memory_request VARCHAR(50),
    memory_limit VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(pod_uid, name)
);

CREATE INDEX IF NOT EXISTS idx_containers_pod_uid ON containers(pod_uid);

CREATE INDEX IF NOT EXISTS idx_containers_namespace_pod_name ON containers(namespace, pod_name);

CREATE INDEX IF NOT EXISTS idx_containers_synced_at ON containers(synced_at);

CREATE TRIGGER update_containers_updated_at
    BEFORE UPDATE ON containers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
