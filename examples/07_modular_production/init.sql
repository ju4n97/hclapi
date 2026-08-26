CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    total_cents INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (id, name, email) VALUES (1, 'Jane Developer', 'jane@company.com') ON CONFLICT DO NOTHING;
INSERT INTO orders (user_id, total_cents, status) VALUES 
    (1, 14900, 'completed'),
    (1, 8900, 'shipped')
ON CONFLICT DO NOTHING;