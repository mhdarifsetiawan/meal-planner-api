package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuRepository interface {
	CreateSelectedMenuAndShoppingList(ctx context.Context, userID string, recipe *model.Recipe, ingredients []model.RecipeIngredient) (*model.SelectMenuResult, error)
	SaveMenuGeneration(ctx context.Context, userID string, options interface{}) error
	GetLatestMenuGenerationToday(ctx context.Context, userID string) (*model.UserMenuGeneration, error)
	GetMenuGenerationsHistory(ctx context.Context, userID string, limit int, offset int) ([]model.UserMenuGeneration, int, error)
}

type pgxMenuRepository struct {
	db *pgxpool.Pool
}

func NewMenuRepository(db *pgxpool.Pool) MenuRepository {
	return &pgxMenuRepository{db: db}
}

func (r *pgxMenuRepository) SaveMenuGeneration(ctx context.Context, userID string, options interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("failed to marshal menu options: %w", err)
	}

	query := `
		INSERT INTO user_menu_generations (user_id, options, generation_date, created_at)
		VALUES ($1, $2::jsonb, CURRENT_DATE, NOW())
	`
	_, err = r.db.Exec(ctx, query, userID, string(optionsJSON))
	if err != nil {
		return fmt.Errorf("failed to save user_menu_generations: %w", err)
	}
	return nil
}

func (r *pgxMenuRepository) GetLatestMenuGenerationToday(ctx context.Context, userID string) (*model.UserMenuGeneration, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, user_id, options, generation_date, created_at
		FROM user_menu_generations
		WHERE user_id = $1 AND generation_date = CURRENT_DATE
		ORDER BY created_at DESC
		LIMIT 1
	`

	var gen model.UserMenuGeneration
	var optionsRaw []byte
	err := r.db.QueryRow(ctx, query, userID).Scan(&gen.ID, &gen.UserID, &optionsRaw, &gen.GenerationDate, &gen.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest menu generation: %w", err)
	}

	_ = json.Unmarshal(optionsRaw, &gen.Options)
	return &gen, nil
}

func (r *pgxMenuRepository) GetMenuGenerationsHistory(ctx context.Context, userID string, limit int, offset int) ([]model.UserMenuGeneration, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM user_menu_generations WHERE user_id = $1`
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count menu generations: %w", err)
	}

	if total == 0 {
		return []model.UserMenuGeneration{}, 0, nil
	}

	query := `
		SELECT id, user_id, options, generation_date, created_at
		FROM user_menu_generations
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query menu generations history: %w", err)
	}
	defer rows.Close()

	var items []model.UserMenuGeneration
	for rows.Next() {
		var item model.UserMenuGeneration
		var optionsRaw []byte
		err := rows.Scan(&item.ID, &item.UserID, &optionsRaw, &item.GenerationDate, &item.CreatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan menu generation row: %w", err)
		}
		_ = json.Unmarshal(optionsRaw, &item.Options)
		items = append(items, item)
	}

	if items == nil {
		items = []model.UserMenuGeneration{}
	}

	return items, total, nil
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

