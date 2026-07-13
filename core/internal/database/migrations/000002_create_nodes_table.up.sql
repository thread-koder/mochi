CREATE TABLE IF NOT EXISTS nodes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL UNIQUE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    internal_ip VARCHAR(50),
    external_ip VARCHAR(50),
    os_image VARCHAR(255) NOT NULL DEFAULT '',
    kernel_version VARCHAR(255) NOT NULL DEFAULT '',
    container_runtime_version VARCHAR(255) NOT NULL DEFAULT '',
    kubelet_version VARCHAR(255) NOT NULL DEFAULT '',
    cpu_capacity VARCHAR(50) NOT NULL DEFAULT '',
    memory_capacity VARCHAR(50) NOT NULL DEFAULT '',
    cpu_allocatable VARCHAR(50) NOT NULL DEFAULT '',
    memory_allocatable VARCHAR(50) NOT NULL DEFAULT '',
    labels JSONB,
    annotations JSONB,
    conditions JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);

CREATE INDEX IF NOT EXISTS idx_nodes_uid ON nodes(uid);

CREATE INDEX IF NOT EXISTS idx_nodes_synced_at ON nodes(synced_at);

CREATE TRIGGER update_nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
