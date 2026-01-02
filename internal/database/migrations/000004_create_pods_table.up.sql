-- Create pods table
CREATE TABLE IF NOT EXISTS pods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    uid VARCHAR(255) NOT NULL UNIQUE,
    node_name VARCHAR(255),
    phase VARCHAR(50) NOT NULL,
    restart_policy VARCHAR(50),
    cpu_request VARCHAR(50),
    cpu_limit VARCHAR(50),
    memory_request VARCHAR(50),
    memory_limit VARCHAR(50),
    labels JSONB,
    annotations JSONB,
    owner_kind VARCHAR(50),
    owner_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on namespace and name for faster lookups
CREATE INDEX IF NOT EXISTS idx_pods_namespace_name ON pods(namespace, name);

-- Create index on uid for unique lookups
CREATE INDEX IF NOT EXISTS idx_pods_uid ON pods(uid);

-- Create index on node_name for node-based queries
CREATE INDEX IF NOT EXISTS idx_pods_node_name ON pods(node_name);

-- Create index on owner for owner-based queries
CREATE INDEX IF NOT EXISTS idx_pods_owner ON pods(owner_kind, owner_name);

-- Create index on synced_at for sync tracking
CREATE INDEX IF NOT EXISTS idx_pods_synced_at ON pods(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_pods_updated_at
    BEFORE UPDATE ON pods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
