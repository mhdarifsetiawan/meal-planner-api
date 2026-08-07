-- Migration 004: Recipes & ingredients

CREATE TABLE recipes (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    goal_tags   JSONB        NOT NULL DEFAULT '[]', -- ["hemat", "sehat"]
    ai_generated BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE recipe_ingredients (
    id              SERIAL       PRIMARY KEY,
    recipe_id       INT          NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    ingredient_name VARCHAR(255) NOT NULL,
    quantity        VARCHAR(50)  NOT NULL,
    unit            VARCHAR(50)
);
