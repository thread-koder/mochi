CREATE TABLE IF NOT EXISTS spans (
    trace_id BYTEA NOT NULL,
    span_id BYTEA NOT NULL,
    parent_span_id BYTEA,
    start_at TIMESTAMP WITH TIME ZONE NOT NULL,
    end_at TIMESTAMP WITH TIME ZONE NOT NULL,
    method VARCHAR(32) NOT NULL,
    route TEXT NOT NULL,
    status_class VARCHAR(8) NOT NULL,
    status_code INT NOT NULL,
    grpc_status INT,
    rpc_system VARCHAR(16) NOT NULL CHECK (rpc_system IN ('http', 'grpc')),
    source VARCHAR(50) NOT NULL DEFAULT 'mochi-ebpf',
    sampled_by VARCHAR(16) NOT NULL CHECK (sampled_by IN ('head', 'error')),
    src_pod_uid VARCHAR(255) NOT NULL,
    src_namespace VARCHAR(255) NOT NULL,
    src_pod VARCHAR(255) NOT NULL,
    dst_pod_uid VARCHAR(255) NOT NULL DEFAULT '',
    dst_namespace VARCHAR(255) NOT NULL DEFAULT '',
    dst_pod VARCHAR(255) NOT NULL DEFAULT '',
    dst_ip VARCHAR(255) NOT NULL,
    dst_port INT NOT NULL,
    actual_dst_ip VARCHAR(255) NOT NULL,
    actual_dst_port INT NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    dst_hostname VARCHAR(255) NOT NULL DEFAULT '',
    from_kind VARCHAR(50),
    from_namespace VARCHAR(255),
    from_name VARCHAR(255),
    to_kind VARCHAR(50),
    to_namespace VARCHAR(255),
    to_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (trace_id, span_id),
    CONSTRAINT spans_trace_id_len CHECK (octet_length(trace_id) = 16),
    CONSTRAINT spans_span_id_len CHECK (octet_length(span_id) = 8),
    CONSTRAINT spans_parent_span_id_len CHECK (parent_span_id IS NULL OR octet_length(parent_span_id) = 8)
);

CREATE INDEX IF NOT EXISTS idx_spans_start_at
    ON spans (start_at);

CREATE INDEX IF NOT EXISTS idx_spans_src_namespace_start
    ON spans (src_namespace, start_at DESC);

CREATE INDEX IF NOT EXISTS idx_spans_dst_namespace_start
    ON spans (dst_namespace, start_at DESC);

CREATE INDEX IF NOT EXISTS idx_spans_from_workload_start
    ON spans (from_kind, from_namespace, from_name, start_at DESC);

CREATE INDEX IF NOT EXISTS idx_spans_to_workload_start
    ON spans (to_kind, to_namespace, to_name, start_at DESC);
