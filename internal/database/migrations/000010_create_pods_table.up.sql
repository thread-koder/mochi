-- Create pods table
CREATE TABLE IF NOT EXISTS pods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    node VARCHAR(255) REFERENCES nodes(name) ON DELETE SET NULL,
    phase VARCHAR(50) NOT NULL,
    restart_policy VARCHAR(50),
    labels JSONB,
    annotations JSONB,
    owner_kind VARCHAR(50),
    owner_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on namespace and name
CREATE INDEX IF NOT EXISTS idx_pods_namespace_name ON pods(namespace, name);

-- Create index on uid
CREATE INDEX IF NOT EXISTS idx_pods_uid ON pods(uid);

-- Create index on node
CREATE INDEX IF NOT EXISTS idx_pods_node ON pods(node);

-- Create index on owner
CREATE INDEX IF NOT EXISTS idx_pods_owner ON pods(owner_kind, owner_name);

-- Create index on synced_at
CREATE INDEX IF NOT EXISTS idx_pods_synced_at ON pods(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_pods_updated_at
    BEFORE UPDATE ON pods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
