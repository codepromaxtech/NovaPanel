-- Multi-server deployment fan-out support

-- Track which server a deployment ran on (nullable for backwards compat)
ALTER TABLE deployments
  ADD COLUMN IF NOT EXISTS server_id UUID REFERENCES servers(id) ON DELETE SET NULL;

-- Per-server result rows for multi-server fan-out deployments
CREATE TABLE IF NOT EXISTS deployment_targets (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  server_id     UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  status        VARCHAR(20) DEFAULT 'pending',
  build_log     TEXT,
  commit_hash   VARCHAR(64),
  started_at    TIMESTAMP,
  completed_at  TIMESTAMP,
  created_at    TIMESTAMP DEFAULT NOW(),
  UNIQUE(deployment_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_deploy_targets_deployment ON deployment_targets(deployment_id);
CREATE INDEX IF NOT EXISTS idx_deploy_targets_server     ON deployment_targets(server_id);
