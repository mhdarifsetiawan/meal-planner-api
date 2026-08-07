package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSubscriptionRepository_GetPlans(t *testing.T) {
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

	repo := NewSubscriptionRepository(dbPool)

	// 1. Get all active plans
	plans, err := repo.GetSubscriptionPlans(ctx)
	if err != nil {
		t.Fatalf("Failed to get subscription plans: %v", err)
	}

	if len(plans) < 2 {
		t.Errorf("Expected at least 2 seeded plans (free & premium), got %d", len(plans))
	}

	// 2. Get plan by name
	freePlan, err := repo.GetSubscriptionPlanByName(ctx, "free")
	if err != nil {
		t.Fatalf("Failed to get free plan: %v", err)
	}
	if freePlan.Name != "free" {
		t.Errorf("Expected plan name 'free', got '%s'", freePlan.Name)
	}

	premiumPlan, err := repo.GetSubscriptionPlanByName(ctx, "premium")
	if err != nil {
		t.Fatalf("Failed to get premium plan: %v", err)
	}
	if premiumPlan.Name != "premium" {
		t.Errorf("Expected plan name 'premium', got '%s'", premiumPlan.Name)
	}

	// 3. Test CreateUserSubscriptionTx
	userRepo := NewUserRepository(dbPool)
	testUserID := "00000000-0000-0000-0000-000000000004"
	nameStr := "Sub Test User"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    testUserID,
		Email: "sub-test@example.com",
		Name:  &nameStr,
		Role:  "user",
	})

	res, err := repo.CreateUserSubscriptionTx(ctx, testUserID, premiumPlan, nil, 29000, "dummy", "dummy_ref_999")
	if err != nil {
		t.Fatalf("Failed to create user subscription tx: %v", err)
	}

	if res.SubscriptionID <= 0 {
		t.Errorf("Expected positive subscription_id, got %d", res.SubscriptionID)
	}

	if res.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", res.Status)
	}
}
