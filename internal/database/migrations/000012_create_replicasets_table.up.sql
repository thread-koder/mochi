-- Create replicasets table
CREATE TABLE IF NOT EXISTS replicasets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    uid VARCHAR(255) NOT NULL UNIQUE,
    replicas INTEGER NOT NULL DEFAULT 0,
    ready_replicas INTEGER NOT NULL DEFAULT 0,
    owner_kind VARCHAR(255),
    owner_name VARCHAR(255),
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

-- Create index on namespace and name for faster lookups
CREATE INDEX IF NOT EXISTS idx_replicasets_namespace_name ON replicasets(namespace, name);

-- Create index on uid for unique lookups
CREATE INDEX IF NOT EXISTS idx_replicasets_uid ON replicasets(uid);

-- Create index on owner for faster lookups by deployment
CREATE INDEX IF NOT EXISTS idx_replicasets_owner ON replicasets(namespace, owner_kind, owner_name);

-- Create index on synced_at for sync tracking
CREATE INDEX IF NOT EXISTS idx_replicasets_synced_at ON replicasets(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_replicasets_updated_at
    BEFORE UPDATE ON replicasets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
