CREATE TABLE deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id),
    commit_hash     VARCHAR(64),
    branch          VARCHAR(100) DEFAULT 'main',
    status          VARCHAR(20) DEFAULT 'pending',
    build_log       TEXT,
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_deployments_app ON deployments(app_id);
