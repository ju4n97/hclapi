CREATE TABLE IF NOT EXISTS members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    tier TEXT NOT NULL DEFAULT 'standard',
    points INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO members (name, email, tier, points) VALUES 
    ('Jane Developer', 'jane@example.com', 'enterprise', 500),
    ('John Doe', 'john@example.com', 'standard', 100);