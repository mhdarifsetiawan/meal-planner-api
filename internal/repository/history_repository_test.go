package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHistoryRepository_GetHistoryByUserID(t *testing.T) {
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

	// 1. Setup mock user and selected menu
	testUserID := "00000000-0000-0000-0000-000000000003"
	userRepo := NewUserRepository(dbPool)
	nameStr := "Test History User"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    testUserID,
		Email: "test-history@example.com",
		Name:  &nameStr,
		Role:  "user",
	})

	menuRepo := NewMenuRepository(dbPool)
	recipe := &model.Recipe{
		Name:                "Nasi Goreng Spesial",
		Description:         "Enak dan Praktis",
		EstimatedTotalPrice: 25000,
	}
	ingredients := []model.RecipeIngredient{
		{Name: "Nasi", Quantity: "2", Unit: "piring", EstimatedPrice: 5000},
		{Name: "Telur", Quantity: "2", Unit: "butir", EstimatedPrice: 6000},
	}

	_, err = menuRepo.CreateSelectedMenuAndShoppingList(ctx, testUserID, recipe, ingredients)
	if err != nil {
		t.Fatalf("Failed to create menu and shopping list: %v", err)
	}

	histRepo := NewHistoryRepository(dbPool)

	// 2. Fetch history
	items, total, err := histRepo.GetHistoryByUserID(ctx, testUserID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if total <= 0 {
		t.Errorf("Expected total > 0, got %d", total)
	}

	if len(items) == 0 {
		t.Fatal("Expected non-empty history items")
	}

	if items[0].RecipeName != "Nasi Goreng Spesial" {
		t.Errorf("Expected recipe_name 'Nasi Goreng Spesial', got '%s'", items[0].RecipeName)
	}
}
