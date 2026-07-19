CREATE TABLE IF NOT EXISTS pods (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    node VARCHAR(255) REFERENCES nodes(name) ON DELETE SET NULL,
    phase VARCHAR(50) NOT NULL,
    restart_policy VARCHAR(50) NOT NULL,
    labels JSONB,
    annotations JSONB,
    owner_kind VARCHAR(50),
    owner_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pods_namespace_name ON pods(namespace, name);

CREATE INDEX IF NOT EXISTS idx_pods_uid ON pods(uid);

CREATE INDEX IF NOT EXISTS idx_pods_node ON pods(node);

CREATE INDEX IF NOT EXISTS idx_pods_owner ON pods(owner_kind, owner_name);

CREATE INDEX IF NOT EXISTS idx_pods_synced_at ON pods(synced_at);

CREATE TRIGGER update_pods_updated_at
    BEFORE UPDATE ON pods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
