package job

import (
	"context"
	"os"
	"testing"

	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConsensusJob_RunConsensusValidation(t *testing.T) {
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

	// 1. Setup mock region, admin and users
	_, _ = dbPool.Exec(ctx, "INSERT INTO provinces (id, name) VALUES (1, 'DKI Jakarta') ON CONFLICT DO NOTHING")
	_, _ = dbPool.Exec(ctx, "INSERT INTO cities (id, province_id, name) VALUES (1, 1, 'Jakarta Selatan') ON CONFLICT DO NOTHING")

	userRepo := repository.NewUserRepository(dbPool)
	adminID := "00000000-0000-0000-0000-000000000010"
	u1ID := "00000000-0000-0000-0000-000000000011"
	u2ID := "00000000-0000-0000-0000-000000000012"
	u3ID := "00000000-0000-0000-0000-000000000013"

	adminName := "Admin Job"
	u1Name := "User 1"
	u2Name := "User 2"
	u3Name := "User 3"

	_ = userRepo.CreateUser(ctx, &model.User{ID: adminID, Email: "admin-job@example.com", Name: &adminName, Role: "admin"})
	_ = userRepo.CreateUser(ctx, &model.User{ID: u1ID, Email: "u1-job@example.com", Name: &u1Name, Role: "user"})
	_ = userRepo.CreateUser(ctx, &model.User{ID: u2ID, Email: "u2-job@example.com", Name: &u2Name, Role: "user"})
	_ = userRepo.CreateUser(ctx, &model.User{ID: u3ID, Email: "u3-job@example.com", Name: &u3Name, Role: "user"})
	// 2. Setup Campaign and Item
	pwRepo := repository.NewPriceWatchRepository(dbPool)
	camp := &model.PriceWatchCampaign{Title: "Job Test Campaign", IsActive: true, CreatedBy: adminID}
	if err := pwRepo.CreateCampaign(ctx, camp); err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	item := &model.PriceWatchItem{CampaignID: camp.ID, IngredientName: "Cabai Merah Job Test", Unit: "kg", DisplayOrder: 1, IsActive: true}
	if err := pwRepo.CreateItem(ctx, item); err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	cityID := 1
	if err := pwRepo.CreateSubmission(ctx, &model.PriceSubmission{WatchItemID: item.ID, UserID: u1ID, CityID: cityID, SubmittedPrice: 50000}); err != nil {
		t.Fatalf("CreateSubmission 1 failed: %v", err)
	}
	if err := pwRepo.CreateSubmission(ctx, &model.PriceSubmission{WatchItemID: item.ID, UserID: u2ID, CityID: cityID, SubmittedPrice: 51000}); err != nil {
		t.Fatalf("CreateSubmission 2 failed: %v", err)
	}
	if err := pwRepo.CreateSubmission(ctx, &model.PriceSubmission{WatchItemID: item.ID, UserID: u3ID, CityID: cityID, SubmittedPrice: 49000}); err != nil {
		t.Fatalf("CreateSubmission 3 failed: %v", err)
	}

	// 3. Execute Consensus Validation Job
	cJob := NewConsensusJob(dbPool)
	summary, err := cJob.RunConsensusValidation(ctx, 3, 15.0)
	if err != nil {
		t.Fatalf("RunConsensusValidation failed: %v", err)
	}

	if summary.ValidatedCount < 3 {
		t.Errorf("Expected at least 3 validated submissions, got %d", summary.ValidatedCount)
	}
}
