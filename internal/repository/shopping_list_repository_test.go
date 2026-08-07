package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestShoppingListRepository_GetAndUpdate(t *testing.T) {
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
	testUserID := "00000000-0000-0000-0000-000000000002"
	userRepo := NewUserRepository(dbPool)
	nameStr := "Test Shopping List User"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    testUserID,
		Email: "test-shopping-list@example.com",
		Name:  &nameStr,
		Role:  "user",
	})

	menuRepo := NewMenuRepository(dbPool)
	recipe := &model.Recipe{
		Name:                "Ayam Goreng Lengkuas",
		Description:         "Lezat",
		EstimatedTotalPrice: 45000,
	}
	ingredients := []model.RecipeIngredient{
		{Name: "Daging Ayam", Quantity: "500", Unit: "gram", EstimatedPrice: 35000},
		{Name: "Lengkuas", Quantity: "1", Unit: "ruas", EstimatedPrice: 2000},
	}

	res, err := menuRepo.CreateSelectedMenuAndShoppingList(ctx, testUserID, recipe, ingredients)
	if err != nil {
		t.Fatalf("Failed to create menu and shopping list: %v", err)
	}

	shopRepo := NewShoppingListRepository(dbPool)

	// 2. Get shopping list by ID
	detail, err := shopRepo.GetShoppingListByID(ctx, res.ShoppingListID, testUserID)
	if err != nil {
		t.Fatalf("Failed to get shopping list: %v", err)
	}

	if detail.RecipeName != "Ayam Goreng Lengkuas" {
		t.Errorf("Expected recipe_name 'Ayam Goreng Lengkuas', got '%s'", detail.RecipeName)
	}
	if len(detail.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(detail.Items))
	}

	// 3. Update checklist item
	updated, err := shopRepo.UpdateShoppingListItemChecklist(ctx, res.ShoppingListID, testUserID, "Daging Ayam", true)
	if err != nil {
		t.Fatalf("Failed to update item checklist: %v", err)
	}

	if !updated.IsChecked {
		t.Errorf("Expected is_checked = true")
	}

	// 4. Verify updated list
	detailAfter, err := shopRepo.GetShoppingListByID(ctx, res.ShoppingListID, testUserID)
	if err != nil {
		t.Fatalf("Failed to get updated shopping list: %v", err)
	}

	if !detailAfter.Items[0].IsChecked {
		t.Errorf("Expected item 0 to be checked in DB")
	}
}
