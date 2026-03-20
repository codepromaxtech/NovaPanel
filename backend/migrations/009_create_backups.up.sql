CREATE TABLE backups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id       UUID REFERENCES servers(id),
    type            VARCHAR(20) NOT NULL DEFAULT 'full',
    storage         VARCHAR(20) NOT NULL DEFAULT 'local',
    path            TEXT,
    size_mb         DECIMAL(10,2),
    status          VARCHAR(20) DEFAULT 'pending',
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    expires_at      TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE backup_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id       UUID REFERENCES servers(id),
    frequency       VARCHAR(20) NOT NULL DEFAULT 'daily',
    retention_days  INTEGER DEFAULT 30,
    type            VARCHAR(20) NOT NULL DEFAULT 'full',
    storage         VARCHAR(20) NOT NULL DEFAULT 'local',
    is_active       BOOLEAN DEFAULT TRUE,
    last_run_at     TIMESTAMP,
    next_run_at     TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);
