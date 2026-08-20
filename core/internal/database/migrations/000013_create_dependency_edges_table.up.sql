CREATE TABLE IF NOT EXISTS dependency_edges (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    from_node_id UUID NOT NULL REFERENCES dependency_nodes(id) ON DELETE CASCADE,
    to_node_id UUID NOT NULL REFERENCES dependency_nodes(id) ON DELETE CASCADE,
    protocol VARCHAR(50) NOT NULL CHECK (protocol IN ('tcp', 'udp')),
    port INT NOT NULL,
    via_service_namespace VARCHAR(255),
    via_service_name VARCHAR(255),
    source VARCHAR(50) NOT NULL DEFAULT 'mochi-ebpf',
    connects DOUBLE PRECISION NOT NULL DEFAULT 0,
    tx_bytes DOUBLE PRECISION NOT NULL DEFAULT 0,
    rx_bytes DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_connections DOUBLE PRECISION NOT NULL DEFAULT 0,
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
    evidence JSONB,
    attrs JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (from_node_id, to_node_id, protocol, port)
);

CREATE INDEX IF NOT EXISTS idx_dependency_edges_last_seen_at
    ON dependency_edges(last_seen_at);

CREATE INDEX IF NOT EXISTS idx_dependency_edges_from_last_seen
    ON dependency_edges(from_node_id, last_seen_at);

CREATE INDEX IF NOT EXISTS idx_dependency_edges_to_last_seen
    ON dependency_edges(to_node_id, last_seen_at);

CREATE INDEX IF NOT EXISTS idx_dependency_edges_via_service
    ON dependency_edges(via_service_namespace, via_service_name);

CREATE TRIGGER update_dependency_edges_updated_at
    BEFORE UPDATE ON dependency_edges
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
