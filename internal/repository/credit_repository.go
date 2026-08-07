package repository

import (
	"context"
	"errors"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInsufficientCredits = errors.New("insufficient credit balance")

type CreditRepository interface {
	GetUserCreditSummary(ctx context.Context, userID string) (*model.UserCreditSummary, error)
	AddCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error
	DeductCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error
}

type pgxCreditRepository struct {
	db *pgxpool.Pool
}

func NewCreditRepository(db *pgxpool.Pool) CreditRepository {
	return &pgxCreditRepository{db: db}
}

func (r *pgxCreditRepository) GetUserCreditSummary(ctx context.Context, userID string) (*model.UserCreditSummary, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	// 1. Get Balance
	var balance int
	balQuery := `SELECT balance FROM user_credits WHERE user_id = $1`
	err := r.db.QueryRow(ctx, balQuery, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			balance = 0
		} else {
			return nil, fmt.Errorf("failed to get user credit balance: %w", err)
		}
	}

	// 2. Get Recent Transactions
	txQuery := `
		SELECT id, user_id, amount, type, reference_id, created_at
		FROM credit_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := r.db.Query(ctx, txQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query credit transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.CreditTransaction
	for rows.Next() {
		var tx model.CreditTransaction
		err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Type, &tx.ReferenceID, &tx.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credit transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if transactions == nil {
		transactions = []model.CreditTransaction{}
	}

	return &model.UserCreditSummary{
		Balance:      balance,
		Transactions: transactions,
	}, nil
}

func (r *pgxCreditRepository) AddCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Upsert balance
	upsertQuery := `
		INSERT INTO user_credits (user_id, balance, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET balance = user_credits.balance + $2, updated_at = NOW()
	`
	if _, err := tx.Exec(ctx, upsertQuery, userID, amount); err != nil {
		return fmt.Errorf("failed to update user credits: %w", err)
	}

	// 2. Record transaction
	recordQuery := `
		INSERT INTO credit_transactions (user_id, amount, type, reference_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	if _, err := tx.Exec(ctx, recordQuery, userID, amount, txType, refID); err != nil {
		return fmt.Errorf("failed to record credit transaction: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *pgxCreditRepository) DeductCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock and check balance
	var currentBalance int
	err = tx.QueryRow(ctx, "SELECT balance FROM user_credits WHERE user_id = $1 FOR UPDATE", userID).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInsufficientCredits
		}
		return fmt.Errorf("failed to fetch user credit balance: %w", err)
	}

	if currentBalance < amount {
		return ErrInsufficientCredits
	}

	// 2. Deduct balance
	deductQuery := `UPDATE user_credits SET balance = balance - $1, updated_at = NOW() WHERE user_id = $2`
	if _, err := tx.Exec(ctx, deductQuery, amount, userID); err != nil {
		return fmt.Errorf("failed to deduct user credits: %w", err)
	}

	// 3. Record transaction
	recordQuery := `
		INSERT INTO credit_transactions (user_id, amount, type, reference_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	if _, err := tx.Exec(ctx, recordQuery, userID, -amount, txType, refID); err != nil {
		return fmt.Errorf("failed to record credit transaction: %w", err)
	}

	return tx.Commit(ctx)
}
