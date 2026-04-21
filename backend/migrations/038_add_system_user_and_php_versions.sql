-- Add system_user column to domains for per-domain OS user isolation
ALTER TABLE domains ADD COLUMN IF NOT EXISTS system_user VARCHAR(32) DEFAULT '';

-- Add php_versions table for tracking installed PHP versions per server
CREATE TABLE IF NOT EXISTS php_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    version VARCHAR(10) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'installing',
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, version)
);

CREATE INDEX IF NOT EXISTS idx_php_versions_server ON php_versions(server_id);
