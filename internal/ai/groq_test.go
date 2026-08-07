package ai

import (
	"context"
	"os"
	"testing"
)

func TestGroqLive(t *testing.T) {
	apiKey := os.Getenv("AI_PROVIDER_API_KEY_GROQ")
	if apiKey == "" {
		t.Skip("AI_PROVIDER_API_KEY_GROQ not set, skipping live test")
	}
	p, err := NewGroqProvider(apiKey, "")
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	params := MenuGenerateParams{
		Goal:          "hemat",
		BudgetAmount:  50000,
		BudgetPeriod:  "harian",
		HouseholdSize: 2,
		Restrictions:  []string{"halal"},
	}

	res, err := p.GenerateMenu(context.Background(), params)
	if err != nil {
		t.Fatalf("GenerateMenu failed: %v", err)
	}

	t.Logf("Success! Generated %d options. First option: %s", len(res.Options), res.Options[0].RecipeName)
}
