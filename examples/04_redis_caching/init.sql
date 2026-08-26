CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    price_cents INT NOT NULL,
    inventory_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO products (sku, name, price_cents, inventory_count)
VALUES 
    ('KB-MECH-01', 'Tactile Mechanical Keyboard', 12900, 42),
    ('MS-OPT-02', 'Ultralight Gaming Mouse', 7900, 115),
    ('MN-4K-03', '27-inch 4K Studio Display', 59900, 18)
ON CONFLICT (sku) DO NOTHING;