-- System-wide settings stored in DB (overrides env vars at runtime)
CREATE TABLE IF NOT EXISTS system_settings (
  key        VARCHAR(100) PRIMARY KEY,
  value      TEXT NOT NULL DEFAULT '',
  encrypted  BOOLEAN DEFAULT FALSE,
  updated_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO system_settings (key, value, encrypted) VALUES
  ('smtp_host',             '', false),
  ('smtp_port',             '587', false),
  ('smtp_user',             '', false),
  ('smtp_password',         '', true),
  ('smtp_from',             '', false),
  ('stripe_secret_key',     '', true),
  ('stripe_webhook_secret', '', true),
  ('stripe_price_enterprise', '', false),
  ('stripe_price_reseller',   '', false)
ON CONFLICT DO NOTHING;
