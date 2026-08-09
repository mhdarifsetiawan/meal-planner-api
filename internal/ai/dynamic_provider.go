package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"
)

type cachedProvider struct {
	provider   AIProvider
	config     *model.AIProviderConfig
	fetchedAt  time.Time
}

type DynamicAIProvider struct {
	configRepo repository.AIConfigRepository
	mu         sync.RWMutex
	cache      *cachedProvider
	cacheTTL   time.Duration
}

func NewDynamicAIProvider(configRepo repository.AIConfigRepository) *DynamicAIProvider {
	return &DynamicAIProvider{
		configRepo: configRepo,
		cacheTTL:   1 * time.Minute,
	}
}

func (d *DynamicAIProvider) resolveActiveProvider(ctx context.Context) (AIProvider, *model.AIProviderConfig, error) {
	d.mu.RLock()
	if d.cache != nil && time.Since(d.cache.fetchedAt) < d.cacheTTL {
		p, c := d.cache.provider, d.cache.config
		d.mu.RUnlock()
		return p, c, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double check cache after acquiring write lock
	if d.cache != nil && time.Since(d.cache.fetchedAt) < d.cacheTTL {
		return d.cache.provider, d.cache.config, nil
	}

	activeConfig, err := d.configRepo.GetActiveConfig(ctx)
	if err != nil || activeConfig == nil {
		log.Printf("Warning: Could not get active AI config from DB (%v), falling back to env var checks", err)
		return d.resolveFromEnv()
	}

	provider, err := d.instantiateProvider(activeConfig.ProviderName, activeConfig.ModelName)
	if err != nil {
		log.Printf("Warning: Failed to instantiate %s (%v), attempting fallback", activeConfig.ProviderName, err)
		return d.resolveFromEnv()
	}

	d.cache = &cachedProvider{
		provider:  provider,
		config:    activeConfig,
		fetchedAt: time.Now(),
	}

	return provider, activeConfig, nil
}

func (d *DynamicAIProvider) instantiateProvider(providerName, modelName string) (AIProvider, error) {
	switch providerName {
	case "openai":
		key := os.Getenv("AI_PROVIDER_API_KEY_OPENAI")
		if key == "" {
			return nil, fmt.Errorf("AI_PROVIDER_API_KEY_OPENAI environment variable is not set")
		}
		return NewOpenAIProvider(key, modelName)
	case "groq":
		key := os.Getenv("AI_PROVIDER_API_KEY_GROQ")
		if key == "" {
			return nil, fmt.Errorf("AI_PROVIDER_API_KEY_GROQ environment variable is not set")
		}
		return NewGroqProvider(key, modelName)
	default:
		return nil, fmt.Errorf("unsupported or unconfigured provider: %s", providerName)
	}
}

func (d *DynamicAIProvider) resolveFromEnv() (AIProvider, *model.AIProviderConfig, error) {
	if openAIKey := os.Getenv("AI_PROVIDER_API_KEY_OPENAI"); openAIKey != "" {
		p, err := NewOpenAIProvider(openAIKey, "")
		if err == nil {
			return p, &model.AIProviderConfig{ProviderName: "openai", ModelName: "gpt-4o-mini"}, nil
		}
	}
	if groqKey := os.Getenv("AI_PROVIDER_API_KEY_GROQ"); groqKey != "" {
		p, err := NewGroqProvider(groqKey, "")
		if err == nil {
			return p, &model.AIProviderConfig{ProviderName: "groq", ModelName: "llama-3.3-70b-versatile"}, nil
		}
	}
	return nil, nil, fmt.Errorf("no AI provider API keys configured in environment variables")
}

func (d *DynamicAIProvider) GenerateMenu(ctx context.Context, params MenuGenerateParams) (*MenuOptions, error) {
	provider, cfg, err := d.resolveActiveProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("AI provider resolution failed: %w", err)
	}

	log.Printf("🤖 Generating menu using active provider: %s (model: %s)", cfg.ProviderName, cfg.ModelName)
	return provider.GenerateMenu(ctx, params)
}
