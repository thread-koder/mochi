-- Create deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    replicas INTEGER NOT NULL DEFAULT 0,
    ready_replicas INTEGER NOT NULL DEFAULT 0,
    available_replicas INTEGER NOT NULL DEFAULT 0,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

-- Create index on namespace and name
CREATE INDEX IF NOT EXISTS idx_deployments_namespace_name ON deployments(namespace, name);

-- Create index on uid
CREATE INDEX IF NOT EXISTS idx_deployments_uid ON deployments(uid);

-- Create index on synced_at
CREATE INDEX IF NOT EXISTS idx_deployments_synced_at ON deployments(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_deployments_updated_at
    BEFORE UPDATE ON deployments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
