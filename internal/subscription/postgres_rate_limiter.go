package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRateLimiter struct {
	db *pgxpool.Pool
}

func NewPostgresRateLimiter(db *pgxpool.Pool) *PostgresRateLimiter {
	return &PostgresRateLimiter{db: db}
}

func (l *PostgresRateLimiter) Allow(ctx context.Context, userID string, maxAllowed int) (bool, int, error) {
	if l.db == nil {
		// If DB pool is nil, allow request gracefully
		return true, 0, nil
	}

	query := `
		SELECT count
		FROM user_daily_generations
		WHERE user_id = $1 AND generation_date = CURRENT_DATE
	`

	var current int
	err := l.db.QueryRow(ctx, query, userID).Scan(&current)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current = 0
		} else {
			return false, 0, fmt.Errorf("failed to query daily rate limit: %w", err)
		}
	}

	if current >= maxAllowed {
		return false, current, nil
	}

	return true, current, nil
}

func (l *PostgresRateLimiter) Increment(ctx context.Context, userID string) error {
	if l.db == nil {
		return nil
	}

	query := `
		INSERT INTO user_daily_generations (user_id, generation_date, count, updated_at)
		VALUES ($1, CURRENT_DATE, 1, NOW())
		ON CONFLICT (user_id, generation_date)
		DO UPDATE SET count = user_daily_generations.count + 1, updated_at = NOW()
	`

	_, err := l.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment daily generation count: %w", err)
	}

	return nil
}
