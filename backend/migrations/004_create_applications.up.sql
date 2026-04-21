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
