-- 002_create_servers.up.sql

CREATE TABLE servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    hostname        VARCHAR(255) NOT NULL,
    ip_address      INET NOT NULL,
    ip_v6_address   INET,
    port            INTEGER DEFAULT 22,
    os              VARCHAR(50),
    role            VARCHAR(30) DEFAULT 'worker',
    status          VARCHAR(20) DEFAULT 'pending',
    agent_version   VARCHAR(20) DEFAULT '',
    agent_status    VARCHAR(20) DEFAULT 'disconnected',
    resources       JSONB DEFAULT '{}',
    allocated       JSONB DEFAULT '{}',
    tags            JSONB DEFAULT '[]',
    last_heartbeat  TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_role ON servers(role);

CREATE TABLE server_metrics (
    id              BIGSERIAL PRIMARY KEY,
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    cpu_percent     DECIMAL(5,2),
    ram_used_mb     BIGINT,
    ram_total_mb    BIGINT,
    disk_used_gb    DECIMAL(10,2),
    disk_total_gb   DECIMAL(10,2),
    load_avg_1      DECIMAL(5,2),
    load_avg_5      DECIMAL(5,2),
    load_avg_15     DECIMAL(5,2),
    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    recorded_at     TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_server_metrics_server_time ON server_metrics(server_id, recorded_at DESC);
