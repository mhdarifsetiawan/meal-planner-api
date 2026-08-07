package repository

import (
	"context"
	"errors"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserPreferenceNotFound = errors.New("user preferences not found")

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	UpsertUserPreferences(ctx context.Context, pref *model.UserPreference) error
	GetUserPreferencesByUserID(ctx context.Context, userID string) (*model.UserPreference, error)
}

type pgxUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgxUserRepository{db: db}
}

func (r *pgxUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, name, city_id, role, created_at)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'user'), NOW())
		ON CONFLICT (id) DO UPDATE 
		SET email = EXCLUDED.email, name = COALESCE(EXCLUDED.name, users.name)
		RETURNING role, created_at
	`
	err := r.db.QueryRow(ctx, query, user.ID, user.Email, user.Name, user.CityID, user.Role).
		Scan(&user.Role, &user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create/upsert user: %w", err)
	}
	return nil
}

func (r *pgxUserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, name, city_id, role, created_at
		FROM users
		WHERE id = $1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.CityID, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return u, nil
}

func (r *pgxUserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, city_id = $4, role = $5
		WHERE id = $1
	`
	cmd, err := r.db.Exec(ctx, query, user.ID, user.Email, user.Name, user.CityID, user.Role)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *pgxUserRepository) UpsertUserPreferences(ctx context.Context, pref *model.UserPreference) error {
	if len(pref.Restrictions) == 0 {
		pref.Restrictions = []byte("[]")
	}

	query := `
		INSERT INTO user_preferences (user_id, goal, budget_amount, budget_period, household_size, restrictions, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET goal = EXCLUDED.goal,
		    budget_amount = EXCLUDED.budget_amount,
		    budget_period = EXCLUDED.budget_period,
		    household_size = EXCLUDED.household_size,
		    restrictions = EXCLUDED.restrictions,
		    updated_at = NOW()
		RETURNING id, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		pref.UserID, pref.Goal, pref.BudgetAmount, pref.BudgetPeriod, pref.HouseholdSize, pref.Restrictions,
	).Scan(&pref.ID, &pref.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert user preferences: %w", err)
	}
	return nil
}

func (r *pgxUserRepository) GetUserPreferencesByUserID(ctx context.Context, userID string) (*model.UserPreference, error) {
	query := `
		SELECT id, user_id, goal, budget_amount, budget_period, household_size, restrictions, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`
	pref := &model.UserPreference{}
	err := r.db.QueryRow(ctx, query, userID).
		Scan(&pref.ID, &pref.UserID, &pref.Goal, &pref.BudgetAmount, &pref.BudgetPeriod, &pref.HouseholdSize, &pref.Restrictions, &pref.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserPreferenceNotFound
		}
		return nil, fmt.Errorf("failed to get user preferences: %w", err)
	}
	return pref, nil
}
