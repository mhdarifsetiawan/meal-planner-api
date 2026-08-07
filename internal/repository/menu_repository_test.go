package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMenuRepository_CreateSelectedMenuAndShoppingList(t *testing.T) {
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

	// Setup mock user in database
	testUserID := "00000000-0000-0000-0000-000000000001"
	userRepo := NewUserRepository(dbPool)
	nameStr := "Test Menu Select"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    testUserID,
		Email: "test-menu-select@example.com",
		Name:  &nameStr,
		Role:  "user",
	})

	menuRepo := NewMenuRepository(dbPool)

	recipe := &model.Recipe{
		Name:                "Tahu Tempe Bacem",
		Description:         "Enak & Hemat",
		EstimatedTotalPrice: 11000,
	}

	ingredients := []model.RecipeIngredient{
		{Name: "Tahu Putih", Quantity: "1", Unit: "papan", EstimatedPrice: 5000},
		{Name: "Tempe", Quantity: "1", Unit: "papan", EstimatedPrice: 6000},
	}

	res, err := menuRepo.CreateSelectedMenuAndShoppingList(ctx, testUserID, recipe, ingredients)
	if err != nil {
		t.Fatalf("Failed to create selected menu and shopping list: %v", err)
	}

	if res.UserMenuID <= 0 {
		t.Errorf("Expected positive user_menu_id, got %d", res.UserMenuID)
	}

	if res.ShoppingListID <= 0 {
		t.Errorf("Expected positive shopping_list_id, got %d", res.ShoppingListID)
	}

	if res.ItemsCount != 2 {
		t.Errorf("Expected 2 items, got %d", res.ItemsCount)
	}
}
