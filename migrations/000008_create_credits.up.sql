-- Migration 008: Credits system

CREATE TABLE user_credits (
    id         SERIAL      PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) UNIQUE,
    balance    INT         NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE credit_transactions (
    id           SERIAL      PRIMARY KEY,
    user_id      UUID        NOT NULL REFERENCES users(id),
    amount       INT         NOT NULL, -- positif (earn) atau negatif (spend)
    type         VARCHAR(30) NOT NULL, -- earn_submission | spend_generate | ...
    reference_id INT,                 -- misal price_submissions.id
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
