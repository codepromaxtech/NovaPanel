CREATE TABLE billing_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    price_cents     INTEGER NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    interval        VARCHAR(20) DEFAULT 'monthly',
    max_domains     INTEGER DEFAULT 10,
    max_databases   INTEGER DEFAULT 5,
    max_email       INTEGER DEFAULT 20,
    disk_gb         INTEGER DEFAULT 10,
    is_active       BOOLEAN DEFAULT TRUE,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    plan_id         UUID REFERENCES billing_plans(id),
    amount_cents    INTEGER NOT NULL,
    currency        VARCHAR(3) DEFAULT 'USD',
    status          VARCHAR(20) DEFAULT 'pending',
    stripe_id       VARCHAR(255),
    paid_at         TIMESTAMP,
    due_at          TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_invoices_user ON invoices(user_id, created_at DESC);
