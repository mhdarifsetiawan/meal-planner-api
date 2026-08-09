package repository

import (
	"context"
	"fmt"
	"strings"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MasterIngredientRepository interface {
	GetAll(ctx context.Context, categoryQuery string, searchQuery string) ([]model.MasterIngredientWithAliases, error)
	GetByID(ctx context.Context, id int) (*model.MasterIngredientWithAliases, error)
	Create(ctx context.Context, req model.CreateMasterIngredientRequest) (*model.MasterIngredientWithAliases, error)
	Update(ctx context.Context, id int, req model.UpdateMasterIngredientRequest) (*model.MasterIngredientWithAliases, error)
	Delete(ctx context.Context, id int) error
	AddAlias(ctx context.Context, ingredientID int, aliasName string) (*model.IngredientAlias, error)
	DeleteAlias(ctx context.Context, aliasID int) error
}

type pgxMasterIngredientRepository struct {
	db *pgxpool.Pool
}

func NewMasterIngredientRepository(db *pgxpool.Pool) MasterIngredientRepository {
	return &pgxMasterIngredientRepository{db: db}
}

func (r *pgxMasterIngredientRepository) GetAll(ctx context.Context, categoryQuery string, searchQuery string) ([]model.MasterIngredientWithAliases, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if categoryQuery != "" && categoryQuery != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(category) = LOWER($%d)", argIndex))
		args = append(args, categoryQuery)
		argIndex++
	}

	if searchQuery != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE $%d OR EXISTS (SELECT 1 FROM ingredient_aliases ia WHERE ia.master_ingredient_id = mi.id AND LOWER(ia.alias_name) LIKE $%d))", argIndex, argIndex))
		args = append(args, "%"+strings.ToLower(searchQuery)+"%")
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT id, category, name, default_unit, created_at, updated_at
		FROM master_ingredients mi
		WHERE %s
		ORDER BY category ASC, name ASC
	`, strings.Join(whereClauses, " AND "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch master ingredients: %w", err)
	}
	defer rows.Close()

	var result []model.MasterIngredientWithAliases
	for rows.Next() {
		var item model.MasterIngredientWithAliases
		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.DefaultUnit, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan master ingredient: %w", err)
		}
		result = append(result, item)
	}

	if result == nil {
		result = []model.MasterIngredientWithAliases{}
	}

	// Fetch all aliases for retrieved items
	for i := range result {
		aliasQuery := `
			SELECT id, master_ingredient_id, alias_name, created_at
			FROM ingredient_aliases
			WHERE master_ingredient_id = $1
			ORDER BY alias_name ASC
		`
		aliasRows, err := r.db.Query(ctx, aliasQuery, result[i].ID)
		if err == nil {
			var aliases []model.IngredientAlias
			for aliasRows.Next() {
				var a model.IngredientAlias
				if err := aliasRows.Scan(&a.ID, &a.MasterIngredientID, &a.AliasName, &a.CreatedAt); err == nil {
					aliases = append(aliases, a)
				}
			}
			aliasRows.Close()
			if aliases == nil {
				aliases = []model.IngredientAlias{}
			}
			result[i].Aliases = aliases
		} else {
			result[i].Aliases = []model.IngredientAlias{}
		}
	}

	return result, nil
}

func (r *pgxMasterIngredientRepository) GetByID(ctx context.Context, id int) (*model.MasterIngredientWithAliases, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, category, name, default_unit, created_at, updated_at
		FROM master_ingredients
		WHERE id = $1
	`
	var item model.MasterIngredientWithAliases
	err := r.db.QueryRow(ctx, query, id).Scan(&item.ID, &item.Category, &item.Name, &item.DefaultUnit, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("master ingredient not found: %w", err)
	}

	// Fetch aliases
	aliasQuery := `
		SELECT id, master_ingredient_id, alias_name, created_at
		FROM ingredient_aliases
		WHERE master_ingredient_id = $1
		ORDER BY alias_name ASC
	`
	aliasRows, err := r.db.Query(ctx, aliasQuery, item.ID)
	var aliases []model.IngredientAlias
	if err == nil {
		for aliasRows.Next() {
			var a model.IngredientAlias
			if err := aliasRows.Scan(&a.ID, &a.MasterIngredientID, &a.AliasName, &a.CreatedAt); err == nil {
				aliases = append(aliases, a)
			}
		}
		aliasRows.Close()
	}
	if aliases == nil {
		aliases = []model.IngredientAlias{}
	}
	item.Aliases = aliases

	return &item, nil
}

func (r *pgxMasterIngredientRepository) Create(ctx context.Context, req model.CreateMasterIngredientRequest) (*model.MasterIngredientWithAliases, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		INSERT INTO master_ingredients (category, name, default_unit, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, category, name, default_unit, created_at, updated_at
	`
	var item model.MasterIngredientWithAliases
	err = tx.QueryRow(ctx, query, req.Category, req.Name, req.DefaultUnit).Scan(
		&item.ID, &item.Category, &item.Name, &item.DefaultUnit, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert master ingredient: %w", err)
	}

	var aliases []model.IngredientAlias
	for _, aliasName := range req.Aliases {
		cleanAlias := strings.TrimSpace(aliasName)
		if cleanAlias == "" {
			continue
		}

		var a model.IngredientAlias
		insertAliasQuery := `
			INSERT INTO ingredient_aliases (master_ingredient_id, alias_name, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (alias_name) DO NOTHING
			RETURNING id, master_ingredient_id, alias_name, created_at
		`
		err := tx.QueryRow(ctx, insertAliasQuery, item.ID, cleanAlias).Scan(&a.ID, &a.MasterIngredientID, &a.AliasName, &a.CreatedAt)
		if err == nil {
			aliases = append(aliases, a)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	if aliases == nil {
		aliases = []model.IngredientAlias{}
	}
	item.Aliases = aliases
	return &item, nil
}

func (r *pgxMasterIngredientRepository) Update(ctx context.Context, id int, req model.UpdateMasterIngredientRequest) (*model.MasterIngredientWithAliases, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE master_ingredients
		SET category = $1, name = $2, default_unit = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, category, name, default_unit, created_at, updated_at
	`
	var item model.MasterIngredientWithAliases
	err := r.db.QueryRow(ctx, query, req.Category, req.Name, req.DefaultUnit, id).Scan(
		&item.ID, &item.Category, &item.Name, &item.DefaultUnit, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update master ingredient: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *pgxMasterIngredientRepository) Delete(ctx context.Context, id int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM master_ingredients WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete master ingredient: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("master ingredient not found")
	}
	return nil
}

func (r *pgxMasterIngredientRepository) AddAlias(ctx context.Context, ingredientID int, aliasName string) (*model.IngredientAlias, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	cleanAlias := strings.TrimSpace(aliasName)
	if cleanAlias == "" {
		return nil, fmt.Errorf("alias name cannot be empty")
	}

	query := `
		INSERT INTO ingredient_aliases (master_ingredient_id, alias_name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, master_ingredient_id, alias_name, created_at
	`
	var a model.IngredientAlias
	err := r.db.QueryRow(ctx, query, ingredientID, cleanAlias).Scan(&a.ID, &a.MasterIngredientID, &a.AliasName, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add alias (might already exist): %w", err)
	}

	return &a, nil
}

func (r *pgxMasterIngredientRepository) DeleteAlias(ctx context.Context, aliasID int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM ingredient_aliases WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, aliasID)
	if err != nil {
		return fmt.Errorf("failed to delete alias: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("alias not found")
	}
	return nil
}
