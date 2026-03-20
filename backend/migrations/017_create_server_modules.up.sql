CREATE TABLE IF NOT EXISTS server_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    module VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(server_id, module)
);
