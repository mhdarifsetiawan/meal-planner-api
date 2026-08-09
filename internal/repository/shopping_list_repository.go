package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrShoppingListNotFound = errors.New("shopping list not found")
var ErrShoppingItemNotFound = errors.New("shopping list item not found")

type ShoppingListRepository interface {
	GetShoppingListByID(ctx context.Context, id int, userID string) (*model.ShoppingListDetail, error)
	UpdateShoppingListItemChecklist(ctx context.Context, id int, userID string, ingredientName string, isChecked bool) (*model.ShoppingItem, error)
	UpdateShoppingListItemPrice(ctx context.Context, id int, userID string, ingredientName string, newPrice int) (*model.ShoppingItem, int, error)
}

type pgxShoppingListRepository struct {
	db *pgxpool.Pool
}

func NewShoppingListRepository(db *pgxpool.Pool) ShoppingListRepository {
	return &pgxShoppingListRepository{db: db}
}

func (r *pgxShoppingListRepository) GetShoppingListByID(ctx context.Context, id int, userID string) (*model.ShoppingListDetail, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT sl.id, sl.meal_selection_id, r.name AS recipe_name, sl.items, ms.total_estimated_price, sl.created_at
		FROM shopping_lists sl
		JOIN meal_selections ms ON ms.id = sl.meal_selection_id
		JOIN recipes r ON r.id = ms.recipe_id
		WHERE sl.id = $1 AND sl.user_id = $2
	`

	var detail model.ShoppingListDetail
	var itemsRaw []byte

	err := r.db.QueryRow(ctx, query, id, userID).
		Scan(&detail.ID, &detail.MealSelectionID, &detail.RecipeName, &itemsRaw, &detail.TotalEstimatedPrice, &detail.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShoppingListNotFound
		}
		return nil, fmt.Errorf("failed to get shopping list by id: %w", err)
	}

	if len(itemsRaw) > 0 {
		if err := json.Unmarshal(itemsRaw, &detail.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal shopping list items: %w", err)
		}
	} else {
		detail.Items = []model.ShoppingItem{}
	}

	return &detail, nil
}

func (r *pgxShoppingListRepository) UpdateShoppingListItemChecklist(
	ctx context.Context,
	id int,
	userID string,
	ingredientName string,
	isChecked bool,
) (*model.ShoppingItem, error) {
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

	querySelect := `
		SELECT items
		FROM shopping_lists
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`

	var itemsRaw []byte
	err = tx.QueryRow(ctx, querySelect, id, userID).Scan(&itemsRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShoppingListNotFound
		}
		return nil, fmt.Errorf("failed to lock shopping list for update: %w", err)
	}

	var items []model.ShoppingItem
	if len(itemsRaw) > 0 {
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}
	}

	var updatedItem *model.ShoppingItem
	found := false
	for i := range items {
		if items[i].IngredientName == ingredientName {
			items[i].IsChecked = isChecked
			updatedItem = &items[i]
			found = true
			break
		}
	}

	if !found {
		return nil, ErrShoppingItemNotFound
	}

	updatedItemsRaw, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated items: %w", err)
	}

	queryUpdate := `
		UPDATE shopping_lists
		SET items = $3::jsonb
		WHERE id = $1 AND user_id = $2
	`
	_, err = tx.Exec(ctx, queryUpdate, id, userID, string(updatedItemsRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to update shopping list items: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return updatedItem, nil
}

func (r *pgxShoppingListRepository) UpdateShoppingListItemPrice(
	ctx context.Context,
	id int,
	userID string,
	ingredientName string,
	newPrice int,
) (*model.ShoppingItem, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Fetch list with items JSONB + ownership
	var itemsRaw []byte
	var listUserID string
	var mealSelectionID int

	err = tx.QueryRow(ctx, `
		SELECT sl.items, sl.user_id, sl.meal_selection_id
		FROM shopping_lists sl
		WHERE sl.id = $1
		FOR UPDATE OF sl
	`, id).Scan(&itemsRaw, &listUserID, &mealSelectionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrShoppingListNotFound
		}
		return nil, 0, fmt.Errorf("failed to lock shopping list for price update: %w", err)
	}
	if listUserID != userID {
		return nil, 0, ErrShoppingListNotFound
	}

	// Unmarshal JSONB items
	var items []model.ShoppingItem
	if len(itemsRaw) > 0 {
		if jsonErr := json.Unmarshal(itemsRaw, &items); jsonErr != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal items: %w", jsonErr)
		}
	}

	// Find and update the target item in the slice
	var updatedItem *model.ShoppingItem
	newTotalEstimatedPrice := 0

	for i := range items {
		if strings.EqualFold(items[i].IngredientName, ingredientName) {
			items[i].EstimatedPrice = newPrice
			updatedItem = &items[i]
		}
		newTotalEstimatedPrice += items[i].EstimatedPrice
	}

	if updatedItem == nil {
		err = ErrShoppingItemNotFound
		return nil, 0, err
	}

	// Marshal updated items back to JSONB
	updatedItemsRaw, jsonErr := json.Marshal(items)
	if jsonErr != nil {
		err = fmt.Errorf("failed to marshal updated items: %w", jsonErr)
		return nil, 0, err
	}

	// Write updated JSONB items back to shopping_lists
	_, err = tx.Exec(ctx, `
		UPDATE shopping_lists SET items = $3::jsonb WHERE id = $1 AND user_id = $2
	`, id, userID, string(updatedItemsRaw))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to update shopping list items: %w", err)
	}

	// Update meal_selections total price
	_, err = tx.Exec(ctx, `
		UPDATE meal_selections SET total_estimated_price = $2 WHERE id = $1
	`, mealSelectionID, newTotalEstimatedPrice)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to update meal selection total price: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit price update transaction: %w", err)
	}

	return updatedItem, newTotalEstimatedPrice, nil
}
