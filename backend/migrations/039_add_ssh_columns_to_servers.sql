-- Add SSH connection columns missing from initial servers migration
ALTER TABLE servers
    ADD COLUMN IF NOT EXISTS ssh_user     VARCHAR(100) DEFAULT 'root',
    ADD COLUMN IF NOT EXISTS ssh_password TEXT         DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_method  VARCHAR(20)  DEFAULT 'password';
