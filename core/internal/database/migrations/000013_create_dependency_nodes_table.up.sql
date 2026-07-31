CREATE TABLE IF NOT EXISTS dependency_nodes (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    kind VARCHAR(50) NOT NULL
        CHECK (kind IN ('Deployment', 'StatefulSet', 'DaemonSet', 'Pod', 'External')),
    namespace VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    metadata JSONB,
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (kind, namespace, name)
);

CREATE INDEX IF NOT EXISTS idx_dependency_nodes_last_seen_at
    ON dependency_nodes(last_seen_at);

CREATE INDEX IF NOT EXISTS idx_dependency_nodes_namespace
    ON dependency_nodes(namespace);

CREATE TRIGGER update_dependency_nodes_updated_at
    BEFORE UPDATE ON dependency_nodes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
