-- 001_create_users.up.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              VARCHAR(255) UNIQUE NOT NULL,
    password_hash      VARCHAR(255) NOT NULL,
    first_name         VARCHAR(100),
    last_name          VARCHAR(100),
    role               VARCHAR(20) NOT NULL DEFAULT 'client',
    status             VARCHAR(20) NOT NULL DEFAULT 'active',
    parent_id          UUID REFERENCES users(id),
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    two_factor_secret  VARCHAR(255),
    last_login_at      TIMESTAMP,
    created_at         TIMESTAMP DEFAULT NOW(),
    updated_at         TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_parent ON users(parent_id);

CREATE TABLE api_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    token_hash  VARCHAR(255) UNIQUE NOT NULL,
    permissions JSONB DEFAULT '[]',
    last_used   TIMESTAMP,
    expires_at  TIMESTAMP,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) UNIQUE NOT NULL,
    ip_address  INET,
    user_agent  TEXT,
    expires_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);
