CREATE TABLE firewall_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    direction       VARCHAR(10) NOT NULL DEFAULT 'in',
    protocol        VARCHAR(10) DEFAULT 'tcp',
    port            VARCHAR(20),
    source_ip       VARCHAR(50) DEFAULT 'any',
    action          VARCHAR(10) NOT NULL DEFAULT 'allow',
    description     VARCHAR(255),
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE security_events (
    id              BIGSERIAL PRIMARY KEY,
    server_id       UUID REFERENCES servers(id),
    event_type      VARCHAR(50) NOT NULL,
    source_ip       VARCHAR(50),
    details         TEXT,
    severity        VARCHAR(20) DEFAULT 'info',
    created_at      TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_security_events_server ON security_events(server_id, created_at DESC);
