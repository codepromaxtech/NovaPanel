CREATE TABLE IF NOT EXISTS transfer_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_server_id    UUID REFERENCES servers(id) ON DELETE SET NULL,
    dest_server_id      UUID REFERENCES servers(id) ON DELETE SET NULL,
    source_path         TEXT NOT NULL,
    dest_path           TEXT NOT NULL,
    direction           VARCHAR(10) DEFAULT 'push',
    rsync_options       TEXT DEFAULT '-avzh --progress',
    exclude_patterns    TEXT DEFAULT '',
    bandwidth_limit     INTEGER DEFAULT 0,
    delete_extra        BOOLEAN DEFAULT FALSE,
    dry_run             BOOLEAN DEFAULT FALSE,
    status              VARCHAR(20) DEFAULT 'pending',
    bytes_transferred   BIGINT DEFAULT 0,
    files_transferred   INTEGER DEFAULT 0,
    progress            INTEGER DEFAULT 0,
    output              TEXT DEFAULT '',
    started_at          TIMESTAMP,
    completed_at        TIMESTAMP,
    created_at          TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transfer_schedules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    source_server_id    UUID REFERENCES servers(id) ON DELETE SET NULL,
    dest_server_id      UUID REFERENCES servers(id) ON DELETE SET NULL,
    source_path         TEXT NOT NULL,
    dest_path           TEXT NOT NULL,
    direction           VARCHAR(10) DEFAULT 'push',
    rsync_options       TEXT DEFAULT '-avzh --progress',
    exclude_patterns    TEXT DEFAULT '',
    bandwidth_limit     INTEGER DEFAULT 0,
    delete_extra        BOOLEAN DEFAULT FALSE,
    cron_expression     VARCHAR(100) NOT NULL,
    is_active           BOOLEAN DEFAULT TRUE,
    last_run            TIMESTAMP,
    next_run            TIMESTAMP,
    created_at          TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfer_jobs_user ON transfer_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_transfer_jobs_status ON transfer_jobs(status);
CREATE INDEX IF NOT EXISTS idx_transfer_schedules_user ON transfer_schedules(user_id);
