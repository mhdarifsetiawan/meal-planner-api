package repository

import (
	"context"
	"errors"
	"fmt"
	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAIConfigNotFound = errors.New("ai provider config not found")

type AIConfigRepository interface {
	GetAllConfigs(ctx context.Context) ([]model.AIProviderConfig, error)
	GetActiveConfig(ctx context.Context) (*model.AIProviderConfig, error)
	SetActiveConfig(ctx context.Context, providerName string) error
	EnsureDefaultConfigs(ctx context.Context) error
}

type pgxAIConfigRepository struct {
	db *pgxpool.Pool
}

func NewAIConfigRepository(db *pgxpool.Pool) AIConfigRepository {
	return &pgxAIConfigRepository{db: db}
}

func (r *pgxAIConfigRepository) EnsureDefaultConfigs(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	defaults := []struct {
		name     string
		model    string
		active   bool
		keyRef   string
		priority int
	}{
		{"openai", "gpt-4o-mini", true, "AI_PROVIDER_API_KEY_OPENAI", 1},
		{"groq", "llama-3.3-70b-versatile", false, "AI_PROVIDER_API_KEY_GROQ", 2},
		{"gemini", "gemini-1.5-flash", false, "AI_PROVIDER_API_KEY_GEMINI", 3},
		{"deepseek", "deepseek-chat-v3", false, "AI_PROVIDER_API_KEY_DEEPSEEK", 4},
	}

	for _, d := range defaults {
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_provider_config WHERE LOWER(provider_name) = LOWER($1))`, d.name).Scan(&exists)
		if err == nil && !exists {
			query := `
				INSERT INTO ai_provider_config (provider_name, model_name, is_active, api_key_ref, priority, created_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
			`
			_, _ = r.db.Exec(ctx, query, d.name, d.model, d.active, d.keyRef, d.priority)
		}
	}

	return nil
}

func (r *pgxAIConfigRepository) GetAllConfigs(ctx context.Context) ([]model.AIProviderConfig, error) {
	if r.db == nil {
		return r.getFallbackConfigs(), nil
	}

	_ = r.EnsureDefaultConfigs(ctx)

	// Automatically clean up duplicate entries if any existed
	cleanupQuery := `
		DELETE FROM ai_provider_config
		WHERE id NOT IN (
			SELECT DISTINCT ON (LOWER(provider_name)) id
			FROM ai_provider_config
			ORDER BY LOWER(provider_name), is_active DESC, id ASC
		);
	`
	_, _ = r.db.Exec(ctx, cleanupQuery)

	query := `
		SELECT id, provider_name, model_name, is_active, api_key_ref, priority, created_at
		FROM ai_provider_config
		ORDER BY priority ASC, id ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ai_provider_config: %w", err)
	}
	defer rows.Close()

	var configs []model.AIProviderConfig
	for rows.Next() {
		var cfg model.AIProviderConfig
		if err := rows.Scan(&cfg.ID, &cfg.ProviderName, &cfg.ModelName, &cfg.IsActive, &cfg.APIKeyRef, &cfg.Priority, &cfg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ai_provider_config row: %w", err)
		}
		r.enrichMetadata(&cfg)
		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return r.getFallbackConfigs(), nil
	}

	return configs, nil
}

func (r *pgxAIConfigRepository) GetActiveConfig(ctx context.Context) (*model.AIProviderConfig, error) {
	if r.db == nil {
		cfg := r.getFallbackConfigs()[0]
		return &cfg, nil
	}

	query := `
		SELECT id, provider_name, model_name, is_active, api_key_ref, priority, created_at
		FROM ai_provider_config
		WHERE is_active = true
		ORDER BY priority ASC
		LIMIT 1
	`
	var cfg model.AIProviderConfig
	err := r.db.QueryRow(ctx, query).Scan(&cfg.ID, &cfg.ProviderName, &cfg.ModelName, &cfg.IsActive, &cfg.APIKeyRef, &cfg.Priority, &cfg.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fallback := r.getFallbackConfigs()[0]
			return &fallback, nil
		}
		return nil, fmt.Errorf("failed to get active ai config: %w", err)
	}

	r.enrichMetadata(&cfg)
	return &cfg, nil
}

func (r *pgxAIConfigRepository) SetActiveConfig(ctx context.Context, providerName string) error {
	if r.db == nil {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set all to inactive
	_, err = tx.Exec(ctx, `UPDATE ai_provider_config SET is_active = false`)
	if err != nil {
		return fmt.Errorf("failed to deactivate all providers: %w", err)
	}

	// Set target provider to active
	cmd, err := tx.Exec(ctx, `UPDATE ai_provider_config SET is_active = true WHERE LOWER(provider_name) = LOWER($1)`, providerName)
	if err != nil {
		return fmt.Errorf("failed to activate provider %s: %w", providerName, err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrAIConfigNotFound
	}

	return tx.Commit(ctx)
}

func (r *pgxAIConfigRepository) enrichMetadata(cfg *model.AIProviderConfig) {
	switch cfg.ProviderName {
	case "openai":
		cfg.Icon = "🟢"
		cfg.Description = "Fast, highly reliable OpenAI model for structured recipe generation."
	case "groq":
		cfg.Icon = "⚡"
		cfg.Description = "Ultra-fast inference speed via Groq LPU hardware."
	case "gemini":
		cfg.Icon = "✨"
		cfg.Description = "Google DeepMind multimodal LLM for food & menu understanding."
	case "deepseek":
		cfg.Icon = "🐋"
		cfg.Description = "High-reasoning cost-effective open weights LLM provider."
	}
}

func (r *pgxAIConfigRepository) getFallbackConfigs() []model.AIProviderConfig {
	return []model.AIProviderConfig{
		{ID: 1, ProviderName: "openai", ModelName: "gpt-4o-mini", IsActive: true, APIKeyRef: "AI_PROVIDER_API_KEY_OPENAI", Priority: 1, Icon: "🟢", Description: "Fast, highly reliable OpenAI model for structured recipe generation."},
		{ID: 2, ProviderName: "groq", ModelName: "llama-3.3-70b-versatile", IsActive: false, APIKeyRef: "AI_PROVIDER_API_KEY_GROQ", Priority: 2, Icon: "⚡", Description: "Ultra-fast inference speed via Groq LPU hardware."},
		{ID: 3, ProviderName: "gemini", ModelName: "gemini-1.5-flash", IsActive: false, APIKeyRef: "AI_PROVIDER_API_KEY_GEMINI", Priority: 3, Icon: "✨", Description: "Google DeepMind multimodal LLM for food & menu understanding."},
		{ID: 4, ProviderName: "deepseek", ModelName: "deepseek-chat-v3", IsActive: false, APIKeyRef: "AI_PROVIDER_API_KEY_DEEPSEEK", Priority: 4, Icon: "🐋", Description: "High-reasoning cost-effective open weights LLM provider."},
	}
}
