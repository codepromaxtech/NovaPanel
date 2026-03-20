-- 004_create_applications.up.sql

CREATE TABLE applications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id       UUID REFERENCES domains(id),
    server_id       UUID REFERENCES servers(id),
    name            VARCHAR(100) NOT NULL,
    app_type        VARCHAR(30),
    runtime         VARCHAR(30),
    deploy_method   VARCHAR(20),
    git_repo        VARCHAR(500),
    git_branch      VARCHAR(100) DEFAULT 'main',
    env_vars        JSONB DEFAULT '{}',
    status          VARCHAR(20) DEFAULT 'pending',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_applications_user ON applications(user_id);
CREATE INDEX idx_applications_domain ON applications(domain_id);

CREATE TABLE deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id  UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    version         VARCHAR(50),
    commit_hash     VARCHAR(64),
    deploy_log      TEXT,
    status          VARCHAR(20) DEFAULT 'pending',
    started_at      TIMESTAMP DEFAULT NOW(),
    finished_at     TIMESTAMP
);

CREATE INDEX idx_deployments_app ON deployments(application_id, started_at DESC);
