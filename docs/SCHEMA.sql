-- ============================================================
-- MasakApa - Database Schema (Reference / Source of Truth)
-- Postgres. Dipakai sebagai acuan untuk golang-migrate files.
-- Sinkron dengan Supabase Auth (auth.users) untuk kolom users.id
-- ============================================================

-- ---------- Region ----------

CREATE TABLE provinces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE cities (
    id SERIAL PRIMARY KEY,
    province_id INT NOT NULL REFERENCES provinces(id),
    name VARCHAR(100) NOT NULL
);

-- ---------- Users & Preferences ----------
-- users.id disamakan dengan auth.users.id dari Supabase Auth (Google OAuth)

CREATE TABLE users (
    id UUID PRIMARY KEY, -- sync dengan Supabase auth.users.id
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    city_id INT REFERENCES cities(id),
    role VARCHAR(20) NOT NULL DEFAULT 'user', -- user | admin
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) UNIQUE,
    goal VARCHAR(20) NOT NULL, -- hemat | sehat | diet | bebas
    budget_amount INT NOT NULL,
    budget_period VARCHAR(20) NOT NULL, -- harian | mingguan
    household_size INT NOT NULL DEFAULT 1,
    restrictions JSONB DEFAULT '[]', -- ["udang", "kacang", ...]
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- AI Provider Config ----------

CREATE TABLE ai_provider_config (
    id SERIAL PRIMARY KEY,
    provider_name VARCHAR(50) NOT NULL, -- openai | groq | gemini | claude
    model_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    api_key_ref VARCHAR(100) NOT NULL, -- nama env var, bukan key asli
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Recipes & Ingredients ----------

CREATE TABLE recipes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    goal_tags JSONB DEFAULT '[]', -- ["hemat", "sehat"]
    ai_generated BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE recipe_ingredients (
    id SERIAL PRIMARY KEY,
    recipe_id INT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    ingredient_name VARCHAR(255) NOT NULL,
    quantity VARCHAR(50) NOT NULL,
    unit VARCHAR(50)
);

-- ---------- Price Data ----------

CREATE TABLE ingredient_price_log (
    id SERIAL PRIMARY KEY,
    ingredient_name VARCHAR(255) NOT NULL,
    city_id INT REFERENCES cities(id),
    price INT NOT NULL,
    source VARCHAR(20) NOT NULL, -- ai_estimate | crowdsource | scrape | api
    confidence_score NUMERIC(3,2), -- 0.00 - 1.00
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_log_lookup ON ingredient_price_log (ingredient_name, city_id, recorded_at DESC);

-- ---------- Meal Selection & Shopping List ----------

CREATE TABLE meal_selections (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    recipe_id INT NOT NULL REFERENCES recipes(id),
    selected_date DATE NOT NULL,
    total_estimated_price INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE shopping_lists (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    meal_selection_id INT NOT NULL REFERENCES meal_selections(id),
    items JSONB NOT NULL, -- [{"name": "...", "qty": "...", "checked": false}]
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Subscription & Monetization ----------

CREATE TABLE subscription_plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL, -- free | premium
    price INT NOT NULL,
    billing_period VARCHAR(20), -- bulanan | tahunan | NULL untuk free
    features JSONB NOT NULL DEFAULT '{}', -- {"max_generate_per_day": 3, "history_access": false, ...}
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE coupons (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    discount_type VARCHAR(20) NOT NULL, -- percent | fixed
    discount_value INT NOT NULL,
    max_uses INT,
    used_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE user_subscriptions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id INT NOT NULL REFERENCES subscription_plans(id),
    coupon_id INT REFERENCES coupons(id),
    status VARCHAR(20) NOT NULL, -- active | expired | canceled
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ends_at TIMESTAMPTZ
);

CREATE TABLE payment_transactions (
    id SERIAL PRIMARY KEY,
    user_subscription_id INT NOT NULL REFERENCES user_subscriptions(id),
    amount INT NOT NULL,
    status VARCHAR(20) NOT NULL, -- pending | success | failed
    payment_gateway VARCHAR(20) NOT NULL DEFAULT 'dummy', -- dummy | wuzzpay
    gateway_ref VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Community Price Watch (Crowdsource) ----------

CREATE TABLE price_watch_campaigns (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE price_watch_items (
    id SERIAL PRIMARY KEY,
    campaign_id INT NOT NULL REFERENCES price_watch_campaigns(id) ON DELETE CASCADE,
    ingredient_name VARCHAR(255) NOT NULL,
    unit VARCHAR(50) NOT NULL,
    icon_url TEXT,
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE price_submissions (
    id SERIAL PRIMARY KEY,
    watch_item_id INT NOT NULL REFERENCES price_watch_items(id),
    user_id UUID NOT NULL REFERENCES users(id),
    city_id INT NOT NULL REFERENCES cities(id),
    submitted_price INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending | validated | rejected
    validated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_submissions_consensus ON price_submissions (watch_item_id, city_id, status, created_at);

-- ---------- Credits ----------

CREATE TABLE user_credits (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) UNIQUE,
    balance INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE credit_transactions (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    amount INT NOT NULL, -- bisa negatif (spend) atau positif (earn)
    type VARCHAR(30) NOT NULL, -- earn_submission | spend_generate | ...
    reference_id INT, -- misal price_submissions.id
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Master Ingredients & Aliases ----------

CREATE TABLE master_ingredients (
    id SERIAL PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL UNIQUE,
    default_unit VARCHAR(50) NOT NULL DEFAULT 'kg',
    baseline_price INT NOT NULL DEFAULT 10000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ingredient_aliases (
    id SERIAL PRIMARY KEY,
    master_ingredient_id INT NOT NULL REFERENCES master_ingredients(id) ON DELETE CASCADE,
    alias_name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_master_ingredients_category ON master_ingredients(category);
CREATE INDEX idx_ingredient_aliases_alias_name ON ingredient_aliases(LOWER(alias_name));

-- ---------- User Daily Rate Limiting & AI Generations ----------

CREATE TABLE user_daily_generations (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    generation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    count INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, generation_date)
);

CREATE TABLE user_menu_generations (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    options JSONB NOT NULL,
    generation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_menu_generations_lookup ON user_menu_generations(user_id, created_at DESC);


