CREATE TABLE IF NOT EXISTS endpoint_slices (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    address_type VARCHAR(50) NOT NULL,
    owner_kind VARCHAR(255),
    owner_name VARCHAR(255),
    endpoints JSONB,
    ports JSONB,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_endpoint_slices_namespace_name ON endpoint_slices(namespace, name);

CREATE INDEX IF NOT EXISTS idx_endpoint_slices_uid ON endpoint_slices(uid);

CREATE INDEX IF NOT EXISTS idx_endpoint_slices_owner ON endpoint_slices(namespace, owner_kind, owner_name);

CREATE INDEX IF NOT EXISTS idx_endpoint_slices_synced_at ON endpoint_slices(synced_at);

CREATE TRIGGER update_endpoint_slices_updated_at
    BEFORE UPDATE ON endpoint_slices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
