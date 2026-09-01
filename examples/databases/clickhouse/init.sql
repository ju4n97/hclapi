CREATE TABLE IF NOT EXISTS analytics_events (
    event_id UUID,
    user_id UInt64,
    path String,
    duration_ms UInt32,
    country LowCardinality(String),
    timestamp DateTime
) ENGINE = MergeTree()
ORDER BY (path, timestamp);

INSERT INTO analytics_events VALUES 
    (generateUUIDv4(), 101, '/api/v1/checkout', 120, 'US', now()),
    (generateUUIDv4(), 102, '/api/v1/checkout', 145, 'US', now()),
    (generateUUIDv4(), 103, '/api/v1/products', 45, 'DE', now());