-- SFTP SSH public key authentication per FTP account
CREATE TABLE IF NOT EXISTS sftp_keys (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ftp_account_id UUID NOT NULL REFERENCES ftp_accounts(id) ON DELETE CASCADE,
  label         VARCHAR(100) NOT NULL DEFAULT 'key',
  public_key    TEXT NOT NULL,
  fingerprint   VARCHAR(128) NOT NULL,
  created_at    TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sftp_keys_account ON sftp_keys(ftp_account_id);
