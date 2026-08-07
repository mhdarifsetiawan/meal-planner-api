package repository

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPriceWatchRepository_CRUD(t *testing.T) {
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

	// Setup mock admin user
	adminUserID := "00000000-0000-0000-0000-000000000005"
	userRepo := NewUserRepository(dbPool)
	adminName := "Admin PriceWatch"
	_ = userRepo.CreateUser(ctx, &model.User{
		ID:    adminUserID,
		Email: "admin-pw@example.com",
		Name:  &adminName,
		Role:  "admin",
	})

	pwRepo := NewPriceWatchRepository(dbPool)

	// 1. Create Campaign
	camp := &model.PriceWatchCampaign{
		Title:       "Pantau Sembako Ramadan",
		Description: "Bantu warga pantau harga sembako",
		IsActive:    true,
		CreatedBy:   adminUserID,
	}

	err = pwRepo.CreateCampaign(ctx, camp)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}
	if camp.ID <= 0 {
		t.Errorf("Expected positive campaign ID, got %d", camp.ID)
	}

	// 2. Create Item
	item := &model.PriceWatchItem{
		CampaignID:     camp.ID,
		IngredientName: "Beras Medium",
		Unit:           "kg",
		DisplayOrder:   1,
		IsActive:       true,
	}

	err = pwRepo.CreateItem(ctx, item)
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
	if item.ID <= 0 {
		t.Errorf("Expected positive item ID, got %d", item.ID)
	}

	// 3. Get Campaign By ID
	fetched, err := pwRepo.GetCampaignByID(ctx, camp.ID)
	if err != nil {
		t.Fatalf("GetCampaignByID failed: %v", err)
	}
	if fetched.Title != "Pantau Sembako Ramadan" {
		t.Errorf("Expected title 'Pantau Sembako Ramadan', got '%s'", fetched.Title)
	}
	if len(fetched.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(fetched.Items))
	}

	// 4. Update Campaign & Item
	camp.Title = "Pantau Sembako Update"
	if err := pwRepo.UpdateCampaign(ctx, camp); err != nil {
		t.Fatalf("UpdateCampaign failed: %v", err)
	}

	item.IngredientName = "Beras Premium"
	if err := pwRepo.UpdateItem(ctx, item); err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}

	// 5. Delete Item & Campaign
	if err := pwRepo.DeleteItem(ctx, item.ID); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	if err := pwRepo.DeleteCampaign(ctx, camp.ID); err != nil {
		t.Fatalf("DeleteCampaign failed: %v", err)
	}
}
