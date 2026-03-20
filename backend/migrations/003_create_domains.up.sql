-- 003_create_domains.up.sql

CREATE TABLE domains (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id        UUID REFERENCES servers(id),
    name             VARCHAR(255) UNIQUE NOT NULL,
    type             VARCHAR(20) DEFAULT 'primary',
    parent_domain_id UUID REFERENCES domains(id),
    document_root    VARCHAR(500),
    web_server       VARCHAR(30) DEFAULT 'nginx',
    php_version      VARCHAR(10),
    ssl_enabled      BOOLEAN DEFAULT FALSE,
    status           VARCHAR(20) DEFAULT 'active',
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_domains_user ON domains(user_id);
CREATE INDEX idx_domains_name ON domains(name);
CREATE INDEX idx_domains_server ON domains(server_id);

CREATE TABLE dns_zones (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    zone_file   TEXT,
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE dns_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id     UUID NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(10) NOT NULL,
    content     TEXT NOT NULL,
    ttl         INTEGER DEFAULT 3600,
    priority    INTEGER,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE ssl_certificates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id       UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    provider        VARCHAR(50) DEFAULT 'letsencrypt',
    certificate     TEXT,
    private_key     TEXT,
    ca_bundle       TEXT,
    issued_at       TIMESTAMP,
    expires_at      TIMESTAMP,
    auto_renew      BOOLEAN DEFAULT TRUE,
    status          VARCHAR(20) DEFAULT 'pending',
    created_at      TIMESTAMP DEFAULT NOW()
);
