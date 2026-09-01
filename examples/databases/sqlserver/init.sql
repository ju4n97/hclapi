IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'hclapi_db')
BEGIN
    CREATE DATABASE hclapi_db;
END
GO

USE hclapi_db;
GO

IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'members')
BEGIN
    CREATE TABLE members (
        id INT IDENTITY(1,1) PRIMARY KEY,
        name NVARCHAR(255) NOT NULL,
        email NVARCHAR(255) UNIQUE NOT NULL,
        tier NVARCHAR(32) NOT NULL DEFAULT 'standard',
        points INT NOT NULL DEFAULT 0,
        created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
    );
END
GO

IF NOT EXISTS (SELECT 1 FROM members WHERE email = 'jane@example.com')
BEGIN
    INSERT INTO members (name, email, tier, points) VALUES ('Jane Developer', 'jane@example.com', 'enterprise', 500);
    INSERT INTO members (name, email, tier, points) VALUES ('John Doe', 'john@example.com', 'standard', 100);
END
GO

CREATE OR ALTER PROCEDURE award_member_points
    @id INT,
    @bonus INT
AS
BEGIN
    SET NOCOUNT ON;
    UPDATE members SET points = points + @bonus WHERE id = @id;
END;
GO