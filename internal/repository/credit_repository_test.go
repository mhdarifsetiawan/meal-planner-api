package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreditRepository_AddAndDeduct(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5434/masakapa?sslmode=disable"
	}

	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to DB: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		t.Skipf("Skipping integration test: DB ping failed: %v", err)
	}

	userRepo := NewUserRepository(dbPool)
	creditRepo := NewCreditRepository(dbPool)

	testUserID := "00000000-0000-0000-0000-000000000099"
	userName := "Credit Test User"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    testUserID,
		Email: "credit-test@example.com",
		Name:  &userName,
		Role:  "user",
	})

	// 1. Add Credit (+10)
	if err := creditRepo.AddCredit(ctx, testUserID, 10, "earn_test", nil); err != nil {
		t.Fatalf("AddCredit failed: %v", err)
	}

	// 2. Get Summary
	summary, err := creditRepo.GetUserCreditSummary(ctx, testUserID)
	if err != nil {
		t.Fatalf("GetUserCreditSummary failed: %v", err)
	}
	if summary.Balance < 10 {
		t.Errorf("Expected balance >= 10, got %d", summary.Balance)
	}

	// 3. Deduct Credit (-3)
	if err := creditRepo.DeductCredit(ctx, testUserID, 3, "spend_test", nil); err != nil {
		t.Fatalf("DeductCredit failed: %v", err)
	}

	// 4. Verify new balance
	summaryAfter, err := creditRepo.GetUserCreditSummary(ctx, testUserID)
	if err != nil {
		t.Fatalf("GetUserCreditSummary failed: %v", err)
	}
	if summaryAfter.Balance != summary.Balance-3 {
		t.Errorf("Expected balance %d, got %d", summary.Balance-3, summaryAfter.Balance)
	}
}
