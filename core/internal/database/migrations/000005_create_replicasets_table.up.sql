CREATE TABLE IF NOT EXISTS replicasets (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_replicasets_namespace_name ON replicasets(namespace, name);

CREATE INDEX IF NOT EXISTS idx_replicasets_uid ON replicasets(uid);

CREATE INDEX IF NOT EXISTS idx_replicasets_owner ON replicasets(namespace, owner_kind, owner_name);

CREATE INDEX IF NOT EXISTS idx_replicasets_synced_at ON replicasets(synced_at);

CREATE TRIGGER update_replicasets_updated_at
    BEFORE UPDATE ON replicasets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
