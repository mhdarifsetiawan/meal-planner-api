-- Migration 005: Price data & meal selections / shopping lists (rollback)

DROP TABLE IF EXISTS shopping_lists;
DROP TABLE IF EXISTS meal_selections;
DROP INDEX IF EXISTS idx_price_log_lookup;
DROP TABLE IF EXISTS ingredient_price_log;
