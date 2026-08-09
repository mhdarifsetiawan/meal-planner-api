package repository

import (
	"context"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HistoryRepository interface {
	GetHistoryByUserID(ctx context.Context, userID string, limit int, offset int) ([]model.HistoryItem, int, error)
	DeleteHistoryItem(ctx context.Context, userID string, historyID int) error
}

type pgxHistoryRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) HistoryRepository {
	return &pgxHistoryRepository{db: db}
}

func (r *pgxHistoryRepository) DeleteHistoryItem(ctx context.Context, userID string, historyID int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Delete associated shopping list if exists
	_, err = tx.Exec(ctx, `DELETE FROM shopping_lists WHERE meal_selection_id = $1 AND user_id = $2`, historyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete associated shopping list: %w", err)
	}

	// 2. Delete meal selection
	cmd, err := tx.Exec(ctx, `DELETE FROM meal_selections WHERE id = $1 AND user_id = $2`, historyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete history item: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("history item not found or unauthorized")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pgxHistoryRepository) GetHistoryByUserID(ctx context.Context, userID string, limit int, offset int) ([]model.HistoryItem, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("database pool is nil")
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 1. Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM meal_selections WHERE user_id = $1`
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to get history count: %w", err)
	}

	if total == 0 {
		return []model.HistoryItem{}, 0, nil
	}

	// 2. Query history items
	query := `
		SELECT ms.id AS meal_selection_id, sl.id AS shopping_list_id, r.id AS recipe_id,
		       r.name AS recipe_name, COALESCE(r.description, ''), ms.selected_date,
		       ms.total_estimated_price, ms.created_at
		FROM meal_selections ms
		JOIN recipes r ON r.id = ms.recipe_id
		LEFT JOIN shopping_lists sl ON sl.meal_selection_id = ms.id
		WHERE ms.user_id = $1
		ORDER BY ms.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query history items: %w", err)
	}
	defer rows.Close()

	var items []model.HistoryItem
	for rows.Next() {
		var item model.HistoryItem
		err := rows.Scan(
			&item.MealSelectionID,
			&item.ShoppingListID,
			&item.RecipeID,
			&item.RecipeName,
			&item.Description,
			&item.SelectedDate,
			&item.TotalEstimatedPrice,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history row: %w", err)
		}
		item.ID = item.MealSelectionID
		items = append(items, item)
	}

	if items == nil {
		items = []model.HistoryItem{}
	}

	return items, total, nil
}
