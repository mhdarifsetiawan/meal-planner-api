-- Migration 000011: Deduplicate ai_provider_config and add unique constraint on provider_name

-- Delete duplicate rows, keeping the active or lowest ID
DELETE FROM ai_provider_config
WHERE id NOT IN (
    SELECT DISTINCT ON (LOWER(provider_name)) id
    FROM ai_provider_config
    ORDER BY LOWER(provider_name), is_active DESC, id ASC
);

-- Create UNIQUE index on provider_name (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_provider_config_unique_name ON ai_provider_config (LOWER(provider_name));
