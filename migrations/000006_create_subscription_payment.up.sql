-- Migration 006: Subscription & monetization

CREATE TABLE subscription_plans (
    id             SERIAL       PRIMARY KEY,
    name           VARCHAR(50)  NOT NULL, -- free | premium
    price          INT          NOT NULL,
    billing_period VARCHAR(20),           -- bulanan | tahunan | NULL untuk free
    features       JSONB        NOT NULL DEFAULT '{}',
        -- e.g. {"max_generate_per_day": 3, "history_access": false}
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE coupons (
    id             SERIAL       PRIMARY KEY,
    code           VARCHAR(50)  NOT NULL UNIQUE,
    discount_type  VARCHAR(20)  NOT NULL, -- percent | fixed
    discount_value INT          NOT NULL,
    max_uses       INT,
    used_count     INT          NOT NULL DEFAULT 0,
    expires_at     TIMESTAMPTZ,
    is_active      BOOLEAN      NOT NULL DEFAULT true
);

CREATE TABLE user_subscriptions (
    id         SERIAL      PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id),
    plan_id    INT         NOT NULL REFERENCES subscription_plans(id),
    coupon_id  INT         REFERENCES coupons(id),
    status     VARCHAR(20) NOT NULL, -- active | expired | canceled
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at    TIMESTAMPTZ
);

CREATE TABLE payment_transactions (
    id                   SERIAL       PRIMARY KEY,
    user_subscription_id INT          NOT NULL REFERENCES user_subscriptions(id),
    amount               INT          NOT NULL,
    status               VARCHAR(20)  NOT NULL, -- pending | success | failed
    payment_gateway      VARCHAR(20)  NOT NULL DEFAULT 'dummy', -- dummy | wuzzpay
    gateway_ref          VARCHAR(255),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Seed: plan Free (harus ada sebagai default)
INSERT INTO subscription_plans (name, price, billing_period, features)
VALUES (
    'free',
    0,
    NULL,
    '{"max_generate_per_day": 3, "history_access": false, "price_watch_submit": false}'
);
