CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL REFERENCES namespaces(name) ON DELETE CASCADE,
    uid VARCHAR(255) NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 0,
    succeeded INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    owner_kind VARCHAR(255),
    owner_name VARCHAR(255),
    labels JSONB,
    annotations JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_jobs_namespace_name ON jobs(namespace, name);

CREATE INDEX IF NOT EXISTS idx_jobs_uid ON jobs(uid);

CREATE INDEX IF NOT EXISTS idx_jobs_owner ON jobs(namespace, owner_kind, owner_name);

CREATE INDEX IF NOT EXISTS idx_jobs_synced_at ON jobs(synced_at);

CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
