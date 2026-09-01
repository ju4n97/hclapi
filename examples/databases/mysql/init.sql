CREATE TABLE IF NOT EXISTS members (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    tier VARCHAR(32) NOT NULL DEFAULT 'standard',
    points INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO members (name, email, tier, points) VALUES 
    ('Jane Developer', 'jane@example.com', 'enterprise', 500),
    ('John Doe', 'john@example.com', 'standard', 100);

DELIMITER //
CREATE PROCEDURE award_member_points(IN p_id INT, IN p_bonus INT)
BEGIN
    UPDATE members SET points = points + p_bonus WHERE id = p_id;
END //
DELIMITER ;