CREATE TABLE IF NOT EXISTS weather_logs (
    id SERIAL PRIMARY KEY,
    city VARCHAR(100) NOT NULL,
    temperature_c NUMERIC(4, 1) NOT NULL,
    condition VARCHAR(100) NOT NULL,
    queried_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);