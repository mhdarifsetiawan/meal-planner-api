-- Migration 009: Seed Premium subscription plan and update Free plan features

INSERT INTO subscription_plans (name, price, billing_period, features)
VALUES (
    'premium',
    29000,
    'bulanan',
    '{"max_generate_per_day": 999, "history_access": true, "price_watch_submit": true, "unlimited_ai": true}'::jsonb
)
ON CONFLICT DO NOTHING;
