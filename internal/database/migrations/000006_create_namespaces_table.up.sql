-- Create namespaces table
CREATE TABLE IF NOT EXISTS namespaces (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    phase VARCHAR(50) NOT NULL,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on name for faster lookups
CREATE INDEX IF NOT EXISTS idx_namespaces_name ON namespaces(name);

-- Create index on uid for unique lookups
CREATE INDEX IF NOT EXISTS idx_namespaces_uid ON namespaces(uid);

-- Create index on synced_at for sync tracking
CREATE INDEX IF NOT EXISTS idx_namespaces_synced_at ON namespaces(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_namespaces_updated_at
    BEFORE UPDATE ON namespaces
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
