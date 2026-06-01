CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    cluster_ip VARCHAR(50),
    ports JSONB,
    selector JSONB,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_services_namespace_name ON services(namespace, name);

CREATE INDEX IF NOT EXISTS idx_services_uid ON services(uid);

CREATE INDEX IF NOT EXISTS idx_services_synced_at ON services(synced_at);

CREATE TRIGGER update_services_updated_at
    BEFORE UPDATE ON services
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
