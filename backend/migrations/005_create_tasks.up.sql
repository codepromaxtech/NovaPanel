-- 005_create_tasks.up.sql

CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            VARCHAR(50) NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}',
    status          VARCHAR(20) DEFAULT 'queued',
    priority        INTEGER DEFAULT 5,
    server_id       UUID REFERENCES servers(id),
    user_id         UUID REFERENCES users(id),
    result          JSONB,
    error           TEXT,
    attempts        INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 3,
    scheduled_at    TIMESTAMP DEFAULT NOW(),
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON tasks(status, priority, scheduled_at);
CREATE INDEX idx_tasks_user ON tasks(user_id);
CREATE INDEX idx_tasks_server ON tasks(server_id);
