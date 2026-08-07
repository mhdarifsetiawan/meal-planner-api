package ai

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNewOpenAIProvider_MissingAPIKey(t *testing.T) {
	// Clear env var temporarily
	origKey := os.Getenv("AI_PROVIDER_API_KEY_OPENAI")
	os.Unsetenv("AI_PROVIDER_API_KEY_OPENAI")
	defer func() {
		if origKey != "" {
			os.Setenv("AI_PROVIDER_API_KEY_OPENAI", origKey)
		}
	}()

	provider, err := NewOpenAIProvider("", "")
	if err == nil {
		t.Errorf("Expected ErrMissingAPIKey error, got nil provider: %v", provider)
	}
}

func TestNewOpenAIProvider_WithKey(t *testing.T) {
	provider, err := NewOpenAIProvider("test-fake-key", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Unexpected error creating provider: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
	if provider.model != "gpt-4o-mini" {
		t.Errorf("Expected model gpt-4o-mini, got %s", provider.model)
	}
}

func TestFormatRestrictions(t *testing.T) {
	res1 := formatRestrictions([]string{})
	if res1 != "Tidak ada" {
		t.Errorf("Expected 'Tidak ada', got '%s'", res1)
	}

	res2 := formatRestrictions([]string{"udang", "kacang"})
	if res2 != "udang, kacang" {
		t.Errorf("Expected 'udang, kacang', got '%s'", res2)
	}
}

func TestMenuOptionsJSONParsing(t *testing.T) {
	rawJSON := `{
		"options": [
			{
				"recipe_name": "Tahu Tempe Bacem & Sayur Bening",
				"description": "Menu hemat bernutrisi tinggi khas Jawa Central",
				"estimated_total_price": 25000,
				"goal_tags": ["hemat", "sehat"],
				"ingredients": [
					{
						"name": "Tahu Putih",
						"quantity": "1",
						"unit": "papan",
						"estimated_price": 5000
					},
					{
						"name": "Tempe",
						"quantity": "1",
						"unit": "papan",
						"estimated_price": 6000
					}
				]
			}
		]
	}`

	var menuOpts MenuOptions
	err := json.Unmarshal([]byte(rawJSON), &menuOpts)
	if err != nil {
		t.Fatalf("Failed to unmarshal menu JSON: %v", err)
	}

	if len(menuOpts.Options) != 1 {
		t.Fatalf("Expected 1 option, got %d", len(menuOpts.Options))
	}

	opt := menuOpts.Options[0]
	if opt.RecipeName != "Tahu Tempe Bacem & Sayur Bening" {
		t.Errorf("Unexpected recipe name: %s", opt.RecipeName)
	}

	if opt.EstimatedTotalPrice != 25000 {
		t.Errorf("Expected 25000, got %d", opt.EstimatedTotalPrice)
	}

	if len(opt.Ingredients) != 2 {
		t.Errorf("Expected 2 ingredients, got %d", len(opt.Ingredients))
	}
}
