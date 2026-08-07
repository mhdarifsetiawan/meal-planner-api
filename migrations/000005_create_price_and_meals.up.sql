-- Migration 005: Price data & meal selections / shopping lists

CREATE TABLE ingredient_price_log (
    id              SERIAL       PRIMARY KEY,
    ingredient_name VARCHAR(255) NOT NULL,
    city_id         INT          REFERENCES cities(id),
    price           INT          NOT NULL,
    source          VARCHAR(20)  NOT NULL, -- ai_estimate | crowdsource | scrape | api
    confidence_score NUMERIC(3,2),          -- 0.00 – 1.00
    recorded_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_log_lookup
    ON ingredient_price_log (ingredient_name, city_id, recorded_at DESC);

CREATE TABLE meal_selections (
    id                    SERIAL      PRIMARY KEY,
    user_id               UUID        NOT NULL REFERENCES users(id),
    recipe_id             INT         NOT NULL REFERENCES recipes(id),
    selected_date         DATE        NOT NULL,
    total_estimated_price INT         NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE shopping_lists (
    id                SERIAL      PRIMARY KEY,
    user_id           UUID        NOT NULL REFERENCES users(id),
    meal_selection_id INT         NOT NULL REFERENCES meal_selections(id),
    items             JSONB       NOT NULL, -- [{"name": "...", "qty": "...", "checked": false}]
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
