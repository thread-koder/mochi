-- Create daemonsets table
CREATE TABLE IF NOT EXISTS daemonsets (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    desired_number_scheduled INTEGER NOT NULL DEFAULT 0,
    number_ready INTEGER NOT NULL DEFAULT 0,
    number_available INTEGER NOT NULL DEFAULT 0,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

-- Create index on namespace and name
CREATE INDEX IF NOT EXISTS idx_daemonsets_namespace_name ON daemonsets(namespace, name);

-- Create index on uid
CREATE INDEX IF NOT EXISTS idx_daemonsets_uid ON daemonsets(uid);

-- Create index on synced_at
CREATE INDEX IF NOT EXISTS idx_daemonsets_synced_at ON daemonsets(synced_at);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_daemonsets_updated_at
    BEFORE UPDATE ON daemonsets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
