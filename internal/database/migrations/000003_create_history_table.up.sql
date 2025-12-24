-- Create history table
CREATE TABLE IF NOT EXISTS history (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN ('recommendation', 'cost', 'resource_change')),
    entity_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('created', 'updated', 'deleted', 'applied')),
    details TEXT, -- JSON string with additional details
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on entity_type and entity_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_history_entity ON history(entity_type, entity_id);

-- Create index on created_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at);

-- Create index on action for filtering
CREATE INDEX IF NOT EXISTS idx_history_action ON history(action);

-- Create index on created_by for user-based queries
CREATE INDEX IF NOT EXISTS idx_history_created_by ON history(created_by);

