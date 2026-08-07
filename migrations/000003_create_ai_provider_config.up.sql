-- Migration 003: AI Provider config table

CREATE TABLE ai_provider_config (
    id            SERIAL       PRIMARY KEY,
    provider_name VARCHAR(50)  NOT NULL, -- openai | groq | gemini | claude
    model_name    VARCHAR(100) NOT NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT false,
    api_key_ref   VARCHAR(100) NOT NULL, -- nama env var, bukan key asli
    priority      INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Seed default OpenAI provider (inactive until API key is configured)
INSERT INTO ai_provider_config (provider_name, model_name, is_active, api_key_ref, priority)
VALUES ('openai', 'gpt-4o-mini', false, 'AI_PROVIDER_API_KEY_OPENAI', 1);
