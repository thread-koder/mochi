-- Create endpoints table
CREATE TABLE IF NOT EXISTS endpoints (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    addresses JSONB,
    ports JSONB,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

-- Create index on namespace and name
CREATE INDEX IF NOT EXISTS idx_endpoints_namespace_name ON endpoints(namespace, name);

-- Create index on uid
CREATE INDEX IF NOT EXISTS idx_endpoints_uid ON endpoints(uid);

-- Create index on synced_at
CREATE INDEX IF NOT EXISTS idx_endpoints_synced_at ON endpoints(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_endpoints_updated_at
    BEFORE UPDATE ON endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
