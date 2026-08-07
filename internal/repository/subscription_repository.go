package repository

import (
	"context"
	"errors"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")

type SubscriptionRepository interface {
	GetSubscriptionPlans(ctx context.Context) ([]model.SubscriptionPlan, error)
	GetSubscriptionPlanByName(ctx context.Context, name string) (*model.SubscriptionPlan, error)
}

type pgxSubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) SubscriptionRepository {
	return &pgxSubscriptionRepository{db: db}
}

func (r *pgxSubscriptionRepository) GetSubscriptionPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, name, price, billing_period, features, is_active, created_at
		FROM subscription_plans
		WHERE is_active = true
		ORDER BY id ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscription plans: %w", err)
	}
	defer rows.Close()

	var plans []model.SubscriptionPlan
	for rows.Next() {
		var p model.SubscriptionPlan
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.BillingPeriod, &p.Features, &p.IsActive, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription plan: %w", err)
		}
		plans = append(plans, p)
	}

	return plans, nil
}

func (r *pgxSubscriptionRepository) GetSubscriptionPlanByName(ctx context.Context, name string) (*model.SubscriptionPlan, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, name, price, billing_period, features, is_active, created_at
		FROM subscription_plans
		WHERE LOWER(name) = LOWER($1) AND is_active = true
		LIMIT 1
	`

	var p model.SubscriptionPlan
	err := r.db.QueryRow(ctx, query, name).
		Scan(&p.ID, &p.Name, &p.Price, &p.BillingPeriod, &p.Features, &p.IsActive, &p.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubscriptionPlanNotFound
		}
		return nil, fmt.Errorf("failed to get subscription plan by name: %w", err)
	}

	return &p, nil
}
