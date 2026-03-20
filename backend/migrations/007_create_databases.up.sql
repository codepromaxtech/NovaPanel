CREATE TABLE databases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id       UUID REFERENCES servers(id),
    name            VARCHAR(100) NOT NULL,
    engine          VARCHAR(20) NOT NULL DEFAULT 'mysql',
    db_user         VARCHAR(100),
    db_password_enc TEXT,
    charset         VARCHAR(30) DEFAULT 'utf8mb4',
    size_mb         DECIMAL(10,2) DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'active',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_databases_name_server ON databases(name, server_id);
