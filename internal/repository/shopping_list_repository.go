package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrShoppingListNotFound = errors.New("shopping list not found")
var ErrShoppingItemNotFound = errors.New("shopping list item not found")

type ShoppingListRepository interface {
	GetShoppingListByID(ctx context.Context, id int, userID string) (*model.ShoppingListDetail, error)
	UpdateShoppingListItemChecklist(ctx context.Context, id int, userID string, ingredientName string, isChecked bool) (*model.ShoppingItem, error)
	UpdateShoppingListItemPrice(ctx context.Context, id int, userID string, ingredientName string, newPrice int, submitToCommunity bool) (*model.ShoppingItem, int, error)
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
	submitToCommunity bool,
) (*model.ShoppingItem, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	querySelect := `
		SELECT sl.items, sl.meal_selection_id, u.city_id
		FROM shopping_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.id = $1 AND sl.user_id = $2
		FOR UPDATE OF sl
	`

	var itemsRaw []byte
	var mealSelectionID int
	var cityID *int

	err = tx.QueryRow(ctx, querySelect, id, userID).Scan(&itemsRaw, &mealSelectionID, &cityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrShoppingListNotFound
		}
		return nil, 0, fmt.Errorf("failed to lock shopping list for price update: %w", err)
	}

	var items []model.ShoppingItem
	if len(itemsRaw) > 0 {
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal items: %w", err)
		}
	}

	var updatedItem *model.ShoppingItem
	found := false
	newTotalEstimatedPrice := 0
	oldPortionPrice := 0

	for i := range items {
		if items[i].IngredientName == ingredientName {
			oldPortionPrice = items[i].EstimatedPrice
			items[i].EstimatedPrice = newPrice
			updatedItem = &items[i]
			found = true
		}
		newTotalEstimatedPrice += items[i].EstimatedPrice
	}

	if !found {
		return nil, 0, ErrShoppingItemNotFound
	}

	updatedItemsRaw, err := json.Marshal(items)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal updated items: %w", err)
	}

	// Update shopping_lists JSON items
	queryUpdateSL := `
		UPDATE shopping_lists
		SET items = $3::jsonb
		WHERE id = $1 AND user_id = $2
	`
	_, err = tx.Exec(ctx, queryUpdateSL, id, userID, string(updatedItemsRaw))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to update shopping list items: %w", err)
	}

	// Update meal_selections total price
	queryUpdateMS := `
		UPDATE meal_selections
		SET total_estimated_price = $2
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, queryUpdateMS, mealSelectionID, newTotalEstimatedPrice)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to update meal selection total price: %w", err)
	}

	// Option B: Local shopping list price edits DO NOT automatically insert into ingredient_price_log.
	// If user explicitly checked "Laporkan ke Pantau Harga Komunitas", submit a pending price_submission.
	if submitToCommunity && cityID != nil {
		unitPriceToLog := newPrice
		if oldPortionPrice > 0 && newPrice > 0 {
			var currentUnitPrice int
			errFetch := tx.QueryRow(ctx, `
				SELECT l.price
				FROM ingredient_price_log l
				WHERE LOWER(l.ingredient_name) = LOWER($1)
				  AND (l.city_id = $2 OR l.city_id IS NULL OR $2 IS NULL)
				ORDER BY CASE WHEN l.source = 'crowdsource' AND l.recorded_at >= NOW() - INTERVAL '7 days' THEN 0 ELSE 1 END, l.recorded_at DESC
				LIMIT 1
			`, ingredientName, cityID).Scan(&currentUnitPrice)
			if errFetch != nil || currentUnitPrice <= 0 {
				_ = tx.QueryRow(ctx, `
					SELECT baseline_price FROM master_ingredients WHERE LOWER(name) = LOWER($1) LIMIT 1
				`, ingredientName).Scan(&currentUnitPrice)
			}

			if currentUnitPrice > 0 {
				ratio := float64(newPrice) / float64(oldPortionPrice)
				unitPriceToLog = int(math.Round(float64(currentUnitPrice) * ratio))
			}
		}

		// Find matching active price_watch_item ID if available
		var watchItemID int
		errWatchItem := tx.QueryRow(ctx, `
			SELECT pwi.id
			FROM price_watch_items pwi
			JOIN price_watch_campaigns pwc ON pwc.id = pwi.campaign_id
			WHERE LOWER(pwi.ingredient_name) = LOWER($1) AND pwc.is_active = true AND pwi.is_active = true
			LIMIT 1
		`, ingredientName).Scan(&watchItemID)

		if errWatchItem == nil && watchItemID > 0 {
			// Submit pending price_submission for consensus validation
			insertSubQuery := `
				INSERT INTO price_submissions (watch_item_id, user_id, city_id, submitted_price, status, created_at)
				VALUES ($1, $2, $3, $4, 'pending', NOW())
			`
			_, _ = tx.Exec(ctx, insertSubQuery, watchItemID, userID, *cityID, unitPriceToLog)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit price update transaction: %w", err)
	}

	return updatedItem, newTotalEstimatedPrice, nil
}
