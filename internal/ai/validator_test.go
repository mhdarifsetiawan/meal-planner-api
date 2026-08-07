package ai

import (
	"errors"
	"testing"
)

func TestValidateMenuOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *MenuOptions
		wantErr error
	}{
		{
			name:    "Nil options",
			opts:    nil,
			wantErr: ErrNilMenuOptions,
		},
		{
			name:    "Empty options list",
			opts:    &MenuOptions{Options: []MenuOption{}},
			wantErr: ErrEmptyMenuOptions,
		},
		{
			name: "Missing recipe name",
			opts: &MenuOptions{
				Options: []MenuOption{
					{RecipeName: "", EstimatedTotalPrice: 10000, Ingredients: []MenuIngredient{{Name: "Bahan"}}},
				},
			},
			wantErr: ErrMissingRecipeName,
		},
		{
			name: "Invalid total price",
			opts: &MenuOptions{
				Options: []MenuOption{
					{RecipeName: "Nasi Goreng", EstimatedTotalPrice: 0, Ingredients: []MenuIngredient{{Name: "Nasi"}}},
				},
			},
			wantErr: ErrInvalidTotalPrice,
		},
		{
			name: "Missing ingredients",
			opts: &MenuOptions{
				Options: []MenuOption{
					{RecipeName: "Nasi Goreng", EstimatedTotalPrice: 15000, Ingredients: []MenuIngredient{}},
				},
			},
			wantErr: ErrMissingIngredients,
		},
		{
			name: "Missing ingredient name",
			opts: &MenuOptions{
				Options: []MenuOption{
					{RecipeName: "Nasi Goreng", EstimatedTotalPrice: 15000, Ingredients: []MenuIngredient{{Name: ""}}},
				},
			},
			wantErr: ErrMissingIngredientName,
		},
		{
			name: "Valid options",
			opts: &MenuOptions{
				Options: []MenuOption{
					{
						RecipeName:          "Soto Ayam",
						Description:         "Enak dan bergizi",
						EstimatedTotalPrice: 30000,
						GoalTags:            []string{"sehat"},
						Ingredients: []MenuIngredient{
							{Name: "Daging Ayam", Quantity: "250", Unit: "gram", EstimatedPrice: 15000},
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMenuOptions(tt.opts)
			if tt.wantErr != nil {
				if err == nil || !errors.Is(err, tt.wantErr) {
					t.Errorf("Expected error containing %v, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
