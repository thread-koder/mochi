CREATE TABLE IF NOT EXISTS cronjobs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    schedule VARCHAR(255) NOT NULL,
    suspend BOOLEAN NOT NULL DEFAULT FALSE,
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_cronjobs_namespace_name ON cronjobs(namespace, name);

CREATE INDEX IF NOT EXISTS idx_cronjobs_uid ON cronjobs(uid);

CREATE INDEX IF NOT EXISTS idx_cronjobs_synced_at ON cronjobs(synced_at);

CREATE TRIGGER update_cronjobs_updated_at
    BEFORE UPDATE ON cronjobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
