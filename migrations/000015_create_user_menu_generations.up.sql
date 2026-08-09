-- Migration 015: Create user_menu_generations table for persistent AI menu options history

CREATE TABLE IF NOT EXISTS user_menu_generations (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options JSONB NOT NULL,
    generation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_menu_generations_lookup ON user_menu_generations(user_id, created_at DESC);
