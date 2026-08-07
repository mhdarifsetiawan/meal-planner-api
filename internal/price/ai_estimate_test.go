package price

import (
	"context"
	"testing"
)

func TestAIEstimateProvider_FallbackCatalog(t *testing.T) {
	provider := NewAIEstimateProvider(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		ingredient    string
		expectedPrice int
	}{
		{
			name:          "Ayam Goreng",
			ingredient:    "Ayam Potong",
			expectedPrice: 35000,
		},
		{
			name:          "Tahu Goreng",
			ingredient:    "Tahu Putih",
			expectedPrice: 5000,
		},
		{
			name:          "Tempe Goreng",
			ingredient:    "Tempe Kedelai",
			expectedPrice: 6000,
		},
		{
			name:          "Unknown Ingredient Fallback",
			ingredient:    "Bumbu Rahasia X",
			expectedPrice: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := provider.GetIngredientPrice(ctx, tt.ingredient, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if res.Price != tt.expectedPrice {
				t.Errorf("Expected price %d, got %d", tt.expectedPrice, res.Price)
			}

			if res.Source != SourceAIEstimate {
				t.Errorf("Expected source 'ai_estimate', got '%s'", res.Source)
			}
		})
	}
}
