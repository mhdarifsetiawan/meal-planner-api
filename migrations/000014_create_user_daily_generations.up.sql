-- Migration 014: Create user_daily_generations table for persistent daily rate limiting

CREATE TABLE IF NOT EXISTS user_daily_generations (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    count INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, generation_date)
);
