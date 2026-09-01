CREATE TABLE IF NOT EXISTS members (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    tier VARCHAR(32) NOT NULL DEFAULT 'standard',
    points INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO members (name, email, tier, points) VALUES 
    ('Jane Developer', 'jane@example.com', 'enterprise', 500),
    ('John Doe', 'john@example.com', 'standard', 100)
ON CONFLICT (email) DO NOTHING;

CREATE OR REPLACE PROCEDURE award_member_points(p_id INT, p_bonus INT)
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE members
    SET points = points + p_bonus
    WHERE id = p_id;
END;
$$;