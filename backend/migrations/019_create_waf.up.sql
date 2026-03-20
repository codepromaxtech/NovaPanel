CREATE TABLE IF NOT EXISTS waf_configs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id         UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE UNIQUE,
    enabled           BOOLEAN DEFAULT FALSE,
    mode              VARCHAR(20) DEFAULT 'detection_only',
    paranoid_level    INTEGER DEFAULT 1,
    allowed_methods   VARCHAR(255) DEFAULT 'GET HEAD POST PUT DELETE',
    max_request_body  INTEGER DEFAULT 13107200,
    crs_enabled       BOOLEAN DEFAULT TRUE,
    sqli_protection   BOOLEAN DEFAULT TRUE,
    xss_protection    BOOLEAN DEFAULT TRUE,
    rfi_protection    BOOLEAN DEFAULT TRUE,
    lfi_protection    BOOLEAN DEFAULT TRUE,
    rce_protection    BOOLEAN DEFAULT TRUE,
    scanner_block     BOOLEAN DEFAULT TRUE,
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS waf_rules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id         UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    rule_id           INTEGER NOT NULL,
    description       TEXT DEFAULT '',
    is_disabled       BOOLEAN DEFAULT TRUE,
    created_at        TIMESTAMP DEFAULT NOW(),
    UNIQUE(server_id, rule_id)
);

CREATE TABLE IF NOT EXISTS waf_whitelist (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id         UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    type              VARCHAR(20) NOT NULL,
    value             VARCHAR(512) NOT NULL,
    reason            TEXT DEFAULT '',
    created_at        TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS waf_logs (
    id                BIGSERIAL PRIMARY KEY,
    server_id         UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    rule_id           INTEGER DEFAULT 0,
    uri               VARCHAR(2048) DEFAULT '',
    client_ip         VARCHAR(45) DEFAULT '',
    message           TEXT DEFAULT '',
    severity          VARCHAR(20) DEFAULT 'warning',
    action            VARCHAR(20) DEFAULT 'blocked',
    created_at        TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_waf_logs_server_id ON waf_logs(server_id);
CREATE INDEX IF NOT EXISTS idx_waf_logs_created_at ON waf_logs(created_at DESC);
