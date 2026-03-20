CREATE TABLE IF NOT EXISTS email_aliases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id       UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    source          VARCHAR(255) NOT NULL,
    destination     VARCHAR(255) NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS email_autoresponders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    subject         VARCHAR(255) NOT NULL,
    body            TEXT NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMP DEFAULT NOW()
);

ALTER TABLE domains ADD COLUMN IF NOT EXISTS catch_all_address VARCHAR(255) DEFAULT '';
