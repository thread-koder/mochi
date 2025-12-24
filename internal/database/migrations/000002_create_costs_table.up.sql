-- Create costs table
CREATE TABLE IF NOT EXISTS costs (
    id BIGSERIAL PRIMARY KEY,
    namespace VARCHAR(255),
    team VARCHAR(255),
    service VARCHAR(255),
    cost_type VARCHAR(50) NOT NULL CHECK (cost_type IN ('compute', 'storage', 'network')),
    amount DECIMAL(15, 2) NOT NULL CHECK (amount >= 0),
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    period VARCHAR(50) NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly')),
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (period_end > period_start)
);

-- Create index on namespace for faster lookups
CREATE INDEX IF NOT EXISTS idx_costs_namespace ON costs(namespace);

-- Create index on team for team-based queries
CREATE INDEX IF NOT EXISTS idx_costs_team ON costs(team);

-- Create index on service for service-based queries
CREATE INDEX IF NOT EXISTS idx_costs_service ON costs(service);

-- Create index on period dates for time-based queries
CREATE INDEX IF NOT EXISTS idx_costs_period ON costs(period_start, period_end);

-- Create index on cost_type for filtering
CREATE INDEX IF NOT EXISTS idx_costs_type ON costs(cost_type);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_costs_updated_at
    BEFORE UPDATE ON costs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

