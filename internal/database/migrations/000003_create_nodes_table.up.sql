-- Create nodes table
CREATE TABLE IF NOT EXISTS nodes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    internal_ip VARCHAR(50),
    external_ip VARCHAR(50),
    os_image VARCHAR(255),
    kernel_version VARCHAR(255),
    container_runtime_version VARCHAR(255),
    kubelet_version VARCHAR(255),
    cpu_capacity VARCHAR(50),
    memory_capacity VARCHAR(50),
    cpu_allocatable VARCHAR(50),
    memory_allocatable VARCHAR(50),
    labels JSONB,
    annotations JSONB,
    conditions JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on name for faster lookups
CREATE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);

-- Create index on uid for unique lookups
CREATE INDEX IF NOT EXISTS idx_nodes_uid ON nodes(uid);

-- Create index on synced_at for sync tracking
CREATE INDEX IF NOT EXISTS idx_nodes_synced_at ON nodes(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_nodes_updated_at
    BEFORE UPDATE ON nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
