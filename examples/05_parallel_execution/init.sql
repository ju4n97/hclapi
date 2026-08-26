CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(32) NOT NULL
);

CREATE TABLE IF NOT EXISTS invoices (
    id SERIAL PRIMARY KEY,
    account_id INT NOT NULL REFERENCES accounts(id),
    amount_cents INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    account_id INT NOT NULL REFERENCES accounts(id),
    action VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO accounts (id, name, tier) VALUES (1, 'Acme Corp', 'enterprise') ON CONFLICT DO NOTHING;
INSERT INTO invoices (account_id, amount_cents, status) VALUES 
    (1, 45000, 'paid'),
    (1, 45000, 'pending')
ON CONFLICT DO NOTHING;
INSERT INTO audit_logs (account_id, action) VALUES 
    (1, 'API_KEY_ROTATED'),
    (1, 'MEMBER_INVITED')
ON CONFLICT DO NOTHING;