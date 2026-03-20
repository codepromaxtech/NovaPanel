CREATE TABLE IF NOT EXISTS setup_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    module          VARCHAR(50) NOT NULL,
    status          VARCHAR(20) DEFAULT 'pending',
    output          TEXT DEFAULT '',
    duration        VARCHAR(20) DEFAULT '',
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_setup_logs_server ON setup_logs(server_id);

-- Add setup_status to servers
ALTER TABLE servers ADD COLUMN IF NOT EXISTS setup_status VARCHAR(20) DEFAULT 'none';
