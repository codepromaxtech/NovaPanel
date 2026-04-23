-- Migration 041: Community + Enterprise tier split, API keys, sessions, alerts, FTP, webhook log, reseller

-- 1. Plan types and feature flags on billing_plans
ALTER TABLE billing_plans
  ADD COLUMN IF NOT EXISTS plan_type VARCHAR(20) DEFAULT 'community'
    CHECK (plan_type IN ('community', 'enterprise', 'reseller')),
  ADD COLUMN IF NOT EXISTS max_servers        INTEGER DEFAULT 1,
  ADD COLUMN IF NOT EXISTS allow_waf          BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_firewall     BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_cloudflare   BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_team         BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_api_keys     BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_k8s          BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_docker       BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_wildcard_ssl BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_ftp          BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_reseller     BOOLEAN DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS allow_multi_deploy BOOLEAN DEFAULT FALSE;

-- 2. Link users to a billing plan + Stripe + TOTP fields
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS plan_id                 UUID REFERENCES billing_plans(id),
  ADD COLUMN IF NOT EXISTS plan_expires_at         TIMESTAMP,
  ADD COLUMN IF NOT EXISTS stripe_customer_id      VARCHAR(255),
  ADD COLUMN IF NOT EXISTS stripe_subscription_id  VARCHAR(255),
  ADD COLUMN IF NOT EXISTS totp_secret             VARCHAR(128),
  ADD COLUMN IF NOT EXISTS totp_backup_codes       TEXT[];

-- 3. API keys table
CREATE TABLE IF NOT EXISTS api_keys (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         VARCHAR(100) NOT NULL,
  key_hash     VARCHAR(64) NOT NULL UNIQUE,
  key_prefix   VARCHAR(12) NOT NULL,
  last_used_at TIMESTAMP,
  expires_at   TIMESTAMP,
  is_active    BOOLEAN DEFAULT TRUE,
  scopes       TEXT[] DEFAULT '{}',
  created_at   TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

-- 4. Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(64) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  used_at    TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pw_reset_hash ON password_reset_tokens(token_hash);

-- 5. Active sessions for session management UI
CREATE TABLE IF NOT EXISTS user_sessions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_jti     VARCHAR(64) NOT NULL UNIQUE,
  ip_address    INET,
  user_agent    TEXT,
  last_seen_at  TIMESTAMP DEFAULT NOW(),
  expires_at    TIMESTAMP NOT NULL,
  created_at    TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_jti  ON user_sessions(token_jti);

-- 6. Alert rules and incidents
CREATE TABLE IF NOT EXISTS alert_rules (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  server_id    UUID REFERENCES servers(id) ON DELETE CASCADE,
  name         VARCHAR(100) NOT NULL,
  metric       VARCHAR(50) NOT NULL,
  threshold    FLOAT NOT NULL,
  operator     VARCHAR(5) DEFAULT '>' CHECK (operator IN ('>', '<', '>=', '<=')),
  duration_min INTEGER DEFAULT 5,
  channel      VARCHAR(20) DEFAULT 'email' CHECK (channel IN ('email', 'webhook', 'slack')),
  destination  TEXT,
  is_active    BOOLEAN DEFAULT TRUE,
  created_at   TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alert_rules_user   ON alert_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_server ON alert_rules(server_id);

CREATE TABLE IF NOT EXISTS alert_incidents (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id     UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
  fired_at    TIMESTAMP DEFAULT NOW(),
  resolved_at TIMESTAMP,
  value       FLOAT,
  notified    BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_alert_incidents_rule ON alert_incidents(rule_id, fired_at DESC);

-- 7. FTP/SFTP accounts
CREATE TABLE IF NOT EXISTS ftp_accounts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  server_id    UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
  username     VARCHAR(64) NOT NULL,
  password_enc TEXT NOT NULL,
  home_dir     TEXT NOT NULL DEFAULT '/var/www',
  quota_mb     INTEGER DEFAULT 1024,
  is_active    BOOLEAN DEFAULT TRUE,
  created_at   TIMESTAMP DEFAULT NOW(),
  UNIQUE(server_id, username)
);
CREATE INDEX IF NOT EXISTS idx_ftp_accounts_user   ON ftp_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_ftp_accounts_server ON ftp_accounts(server_id);

-- 8. Webhook delivery log
CREATE TABLE IF NOT EXISTS webhook_deliveries (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_id        UUID REFERENCES applications(id) ON DELETE CASCADE,
  event         VARCHAR(50),
  payload       JSONB,
  response_code INTEGER,
  response_body TEXT,
  duration_ms   INTEGER,
  delivered_at  TIMESTAMP DEFAULT NOW(),
  success       BOOLEAN DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_app ON webhook_deliveries(app_id, delivered_at DESC);

-- 9. Env vars encryption column on applications
ALTER TABLE applications
  ADD COLUMN IF NOT EXISTS env_vars_enc TEXT;

-- 10. Reseller allocations
CREATE TABLE IF NOT EXISTS reseller_quotas (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reseller_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  max_domains   INTEGER DEFAULT 5,
  max_databases INTEGER DEFAULT 2,
  max_email     INTEGER DEFAULT 10,
  disk_gb       INTEGER DEFAULT 5,
  created_at    TIMESTAMP DEFAULT NOW(),
  UNIQUE(reseller_id, client_id)
);
CREATE INDEX IF NOT EXISTS idx_reseller_quotas_reseller ON reseller_quotas(reseller_id);
CREATE INDEX IF NOT EXISTS idx_reseller_quotas_client   ON reseller_quotas(client_id);

-- 11. DKIM record storage on email domains
ALTER TABLE email_domains
  ADD COLUMN IF NOT EXISTS dkim_record TEXT;

-- 12. Missing performance indexes
CREATE INDEX IF NOT EXISTS idx_email_accounts_domain ON email_accounts(domain_id);
CREATE INDEX IF NOT EXISTS idx_backups_server        ON backups(server_id);
CREATE INDEX IF NOT EXISTS idx_deployments_app       ON deployments(app_id);
CREATE INDEX IF NOT EXISTS idx_docker_stacks_server  ON docker_stacks(server_id);

-- 13. Seed default community / enterprise / reseller plans
INSERT INTO billing_plans (
  name, price_cents, plan_type, max_servers, max_domains, max_databases, max_email, disk_gb,
  allow_waf, allow_firewall, allow_cloudflare, allow_team, allow_api_keys,
  allow_k8s, allow_docker, allow_wildcard_ssl, allow_ftp, allow_reseller, allow_multi_deploy
) VALUES
  ('Community', 0, 'community', 1, 3, 2, 10, 5,
   FALSE, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE, FALSE),
  ('Enterprise', 4900, 'enterprise', 20, 100, 50, 500, 500,
   TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, FALSE, TRUE),
  ('Reseller', 9900, 'reseller', 50, 500, 200, 2000, 2000,
   TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE)
ON CONFLICT DO NOTHING;
