package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSubscriptionPlanNotFound = errors.New("subscription plan not found")

type SubscriptionRepository interface {
	GetSubscriptionPlans(ctx context.Context) ([]model.SubscriptionPlan, error)
	GetSubscriptionPlanByName(ctx context.Context, name string) (*model.SubscriptionPlan, error)
	CreateUserSubscriptionTx(ctx context.Context, userID string, plan *model.SubscriptionPlan, couponID *int, amount int, gateway string, gatewayRef string) (*model.UserSubscriptionResult, error)
	GetUserDailyLimit(ctx context.Context, userID string) (int, error)
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

func (r *pgxSubscriptionRepository) CreateUserSubscriptionTx(
	ctx context.Context,
	userID string,
	plan *model.SubscriptionPlan,
	couponID *int,
	amount int,
	gateway string,
	gatewayRef string,
) (*model.UserSubscriptionResult, error) {
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

	// 1. Deactivate old active subscriptions
	cancelQuery := `UPDATE user_subscriptions SET status = 'canceled' WHERE user_id = $1 AND status = 'active'`
	_, _ = tx.Exec(ctx, cancelQuery, userID)

	// 2. Insert into user_subscriptions
	var subID int
	var startedAt time.Time
	var endsAt *time.Time

	var subQuery string
	if plan.BillingPeriod != nil && *plan.BillingPeriod == "bulanan" {
		subQuery = `
			INSERT INTO user_subscriptions (user_id, plan_id, coupon_id, status, started_at, ends_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW() + INTERVAL '30 days')
			RETURNING id, started_at, ends_at
		`
	} else {
		subQuery = `
			INSERT INTO user_subscriptions (user_id, plan_id, coupon_id, status, started_at, ends_at)
			VALUES ($1, $2, $3, 'active', NOW(), NULL)
			RETURNING id, started_at, ends_at
		`
	}

	err = tx.QueryRow(ctx, subQuery, userID, plan.ID, couponID).Scan(&subID, &startedAt, &endsAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user_subscription: %w", err)
	}

	// 3. Insert into payment_transactions
	txQuery := `
		INSERT INTO payment_transactions (user_subscription_id, amount, status, payment_gateway, gateway_ref, created_at)
		VALUES ($1, $2, 'success', $3, $4, NOW())
	`
	_, err = tx.Exec(ctx, txQuery, subID, amount, gateway, gatewayRef)
	if err != nil {
		return nil, fmt.Errorf("failed to insert payment_transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit subscription transaction: %w", err)
	}

	return &model.UserSubscriptionResult{
		SubscriptionID: subID,
		PlanName:       plan.Name,
		Status:         "active",
		Amount:         amount,
		PaymentGateway: gateway,
		GatewayRef:     gatewayRef,
		StartedAt:      startedAt,
		EndsAt:         endsAt,
	}, nil
}

func (r *pgxSubscriptionRepository) GetUserDailyLimit(ctx context.Context, userID string) (int, error) {
	if r.db == nil {
		return 3, nil
	}

	query := `
		SELECT (sp.features->>'max_generate_per_day')::int
		FROM user_subscriptions us
		JOIN subscription_plans sp ON sp.id = us.plan_id
		WHERE us.user_id = $1 AND us.status = 'active' AND (us.ends_at IS NULL OR us.ends_at > NOW())
		ORDER BY us.started_at DESC
		LIMIT 1
	`

	var limit int
	err := r.db.QueryRow(ctx, query, userID).Scan(&limit)
	if err == nil && limit > 0 {
		return limit, nil
	}

	// Fallback to free plan feature limit
	freeQuery := `
		SELECT (features->>'max_generate_per_day')::int
		FROM subscription_plans
		WHERE LOWER(name) = 'free' AND is_active = true
		LIMIT 1
	`
	_ = r.db.QueryRow(ctx, freeQuery).Scan(&limit)
	if limit <= 0 {
		limit = 3
	}

	return limit, nil
}
