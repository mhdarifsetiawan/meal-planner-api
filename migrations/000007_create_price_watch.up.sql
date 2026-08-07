-- Migration 007: Community Price Watch (Crowdsource)

CREATE TABLE price_watch_campaigns (
    id          SERIAL       PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_by  UUID         NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE price_watch_items (
    id              SERIAL       PRIMARY KEY,
    campaign_id     INT          NOT NULL REFERENCES price_watch_campaigns(id) ON DELETE CASCADE,
    ingredient_name VARCHAR(255) NOT NULL,
    unit            VARCHAR(50)  NOT NULL,
    icon_url        TEXT,
    display_order   INT          NOT NULL DEFAULT 0,
    is_active       BOOLEAN      NOT NULL DEFAULT true
);

CREATE TABLE price_submissions (
    id              SERIAL      PRIMARY KEY,
    watch_item_id   INT         NOT NULL REFERENCES price_watch_items(id),
    user_id         UUID        NOT NULL REFERENCES users(id),
    city_id         INT         NOT NULL REFERENCES cities(id),
    submitted_price INT         NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | validated | rejected
    validated_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_submissions_consensus
    ON price_submissions (watch_item_id, city_id, status, created_at);
