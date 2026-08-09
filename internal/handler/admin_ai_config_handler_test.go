package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type MockAIConfigRepository struct {
	configs []model.AIProviderConfig
}

func NewMockAIConfigRepository() *MockAIConfigRepository {
	return &MockAIConfigRepository{
		configs: []model.AIProviderConfig{
			{ID: 1, ProviderName: "openai", ModelName: "gpt-4o-mini", IsActive: true},
			{ID: 2, ProviderName: "groq", ModelName: "llama-3.3-70b-versatile", IsActive: false},
		},
	}
}

func (m *MockAIConfigRepository) EnsureDefaultConfigs(ctx context.Context) error {
	return nil
}

func (m *MockAIConfigRepository) GetAllConfigs(ctx context.Context) ([]model.AIProviderConfig, error) {
	return m.configs, nil
}

func (m *MockAIConfigRepository) GetActiveConfig(ctx context.Context) (*model.AIProviderConfig, error) {
	for _, c := range m.configs {
		if c.IsActive {
			return &c, nil
		}
	}
	return &m.configs[0], nil
}

func (m *MockAIConfigRepository) SetActiveConfig(ctx context.Context, providerName string) error {
	found := false
	for i := range m.configs {
		if m.configs[i].ProviderName == providerName {
			m.configs[i].IsActive = true
			found = true
		} else {
			m.configs[i].IsActive = false
		}
	}
	if !found {
		return repository.ErrAIConfigNotFound
	}
	return nil
}

func TestAdminAIConfigHandler_GetConfigs(t *testing.T) {
	repo := NewMockAIConfigRepository()
	h := NewAdminAIConfigHandler(repo)

	app := fiber.New()
	app.Get("/admin/ai/configs", h.HandleGetConfigs)

	req := httptest.NewRequest("GET", "/admin/ai/configs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAdminAIConfigHandler_SelectConfig(t *testing.T) {
	repo := NewMockAIConfigRepository()
	h := NewAdminAIConfigHandler(repo)

	app := fiber.New()
	app.Post("/admin/ai/configs/select", h.HandleSelectConfig)

	// Test valid selection
	body, _ := json.Marshal(map[string]string{"provider_name": "groq"})
	req := httptest.NewRequest("POST", "/admin/ai/configs/select", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	active, _ := repo.GetActiveConfig(context.Background())
	if active.ProviderName != "groq" {
		t.Errorf("Expected active provider to be 'groq', got '%s'", active.ProviderName)
	}
}
