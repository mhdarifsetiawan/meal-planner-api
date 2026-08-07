-- Migration 002: Users & preferences
-- users.id synced with Supabase auth.users.id (Google OAuth)

CREATE TABLE users (
    id         UUID         PRIMARY KEY,          -- sync dengan Supabase auth.users.id
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255),
    city_id    INT          REFERENCES cities(id),
    role       VARCHAR(20)  NOT NULL DEFAULT 'user', -- user | admin
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE user_preferences (
    id              SERIAL       PRIMARY KEY,
    user_id         UUID         NOT NULL REFERENCES users(id) UNIQUE,
    goal            VARCHAR(20)  NOT NULL,          -- hemat | sehat | diet | bebas
    budget_amount   INT          NOT NULL,
    budget_period   VARCHAR(20)  NOT NULL,          -- harian | mingguan
    household_size  INT          NOT NULL DEFAULT 1,
    restrictions    JSONB        NOT NULL DEFAULT '[]', -- ["udang", "kacang", ...]
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
