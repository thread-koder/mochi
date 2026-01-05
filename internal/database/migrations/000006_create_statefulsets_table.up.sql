-- Create statefulsets table
CREATE TABLE IF NOT EXISTS statefulsets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    uid VARCHAR(255) NOT NULL UNIQUE,
    replicas INTEGER NOT NULL DEFAULT 0,
    ready_replicas INTEGER NOT NULL DEFAULT 0,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

-- Create index on namespace and name for faster lookups
CREATE INDEX IF NOT EXISTS idx_statefulsets_namespace_name ON statefulsets(namespace, name);

-- Create index on uid for unique lookups
CREATE INDEX IF NOT EXISTS idx_statefulsets_uid ON statefulsets(uid);

-- Create index on synced_at for sync tracking
CREATE INDEX IF NOT EXISTS idx_statefulsets_synced_at ON statefulsets(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_statefulsets_updated_at
    BEFORE UPDATE ON statefulsets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
