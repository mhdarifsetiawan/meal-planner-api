package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuRepository interface {
	CreateSelectedMenuAndShoppingList(ctx context.Context, userID string, recipe *model.Recipe, ingredients []model.RecipeIngredient) (*model.SelectMenuResult, error)
}

type pgxMenuRepository struct {
	db *pgxpool.Pool
}

func NewMenuRepository(db *pgxpool.Pool) MenuRepository {
	return &pgxMenuRepository{db: db}
}

func (r *pgxMenuRepository) CreateSelectedMenuAndShoppingList(
	ctx context.Context,
	userID string,
	recipe *model.Recipe,
	ingredients []model.RecipeIngredient,
) (*model.SelectMenuResult, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Insert into recipes
	var recipeID int
	queryRecipe := `
		INSERT INTO recipes (name, description, goal_tags, ai_generated, created_at)
		VALUES ($1, $2, '[]'::jsonb, true, NOW())
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryRecipe, recipe.Name, recipe.Description).Scan(&recipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert recipe: %w", err)
	}

	// 2. Insert into recipe_ingredients
	for _, ing := range ingredients {
		queryIng := `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_name, quantity, unit)
			VALUES ($1, $2, $3, $4)
		`
		_, err = tx.Exec(ctx, queryIng, recipeID, ing.Name, ing.Quantity, ing.Unit)
		if err != nil {
			return nil, fmt.Errorf("failed to insert recipe ingredient: %w", err)
		}
	}

	// 3. Insert into meal_selections
	var mealSelectionID int
	queryMealSelection := `
		INSERT INTO meal_selections (user_id, recipe_id, selected_date, total_estimated_price, created_at)
		VALUES ($1, $2, CURRENT_DATE, $3, NOW())
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryMealSelection, userID, recipeID, recipe.EstimatedTotalPrice).Scan(&mealSelectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert meal_selections: %w", err)
	}

	// 4. Construct items JSON for shopping_lists
	type shoppingItem struct {
		IngredientName string `json:"ingredient_name"`
		Quantity       string `json:"quantity"`
		Unit           string `json:"unit"`
		EstimatedPrice int    `json:"estimated_price"`
		IsChecked      bool   `json:"is_checked"`
	}

	var items []shoppingItem
	for _, ing := range ingredients {
		items = append(items, shoppingItem{
			IngredientName: ing.Name,
			Quantity:       ing.Quantity,
			Unit:           ing.Unit,
			EstimatedPrice: ing.EstimatedPrice,
			IsChecked:      false,
		})
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal shopping list items: %w", err)
	}

	// 5. Insert into shopping_lists
	var shoppingListID int
	queryShoppingList := `
		INSERT INTO shopping_lists (user_id, meal_selection_id, items, created_at)
		VALUES ($1, $2, $3::jsonb, NOW())
		RETURNING id
	`
	err = tx.QueryRow(ctx, queryShoppingList, userID, mealSelectionID, string(itemsJSON)).Scan(&shoppingListID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert shopping_list: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &model.SelectMenuResult{
		UserMenuID:          mealSelectionID,
		ShoppingListID:      shoppingListID,
		RecipeName:          recipe.Name,
		TotalEstimatedPrice: recipe.EstimatedTotalPrice,
		ItemsCount:          len(ingredients),
	}, nil
}
