ALTER TABLE domains ADD COLUMN IF NOT EXISTS is_load_balancer BOOLEAN DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS domain_backend_servers (
    domain_id UUID REFERENCES domains(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    weight INTEGER DEFAULT 1,
    PRIMARY KEY (domain_id, server_id)
);
