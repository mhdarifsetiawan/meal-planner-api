package ai

import (
	"errors"
	"fmt"
)

var (
	ErrNilMenuOptions       = errors.New("AI menu options response is nil")
	ErrEmptyMenuOptions     = errors.New("AI response contains no menu options")
	ErrMissingRecipeName    = errors.New("AI recipe is missing recipe_name")
	ErrInvalidTotalPrice    = errors.New("AI recipe total price must be greater than 0")
	ErrMissingIngredients   = errors.New("AI recipe has no ingredients list")
	ErrMissingIngredientName = errors.New("AI ingredient is missing name")
)

// ValidateMenuOptions validates the structure and content of AI generated menu options.
func ValidateMenuOptions(opts *MenuOptions) error {
	if opts == nil {
		return ErrNilMenuOptions
	}

	if len(opts.Options) == 0 {
		return ErrEmptyMenuOptions
	}

	for i, opt := range opts.Options {
		if opt.RecipeName == "" {
			return fmt.Errorf("option %d: %w", i+1, ErrMissingRecipeName)
		}

		if opt.EstimatedTotalPrice <= 0 {
			return fmt.Errorf("option %d (%s): %w", i+1, opt.RecipeName, ErrInvalidTotalPrice)
		}

		if len(opt.Ingredients) == 0 {
			return fmt.Errorf("option %d (%s): %w", i+1, opt.RecipeName, ErrMissingIngredients)
		}

		for j, ing := range opt.Ingredients {
			if ing.Name == "" {
				return fmt.Errorf("option %d (%s) ingredient %d: %w", i+1, opt.RecipeName, j+1, ErrMissingIngredientName)
			}
		}
	}

	return nil
}
