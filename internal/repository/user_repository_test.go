package repository

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"meal-planner-api/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDBPool(t *testing.T) *pgxpool.Pool {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5434/masakapa?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping integration test; failed to connect to Postgres DB: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping integration test; DB ping failed: %v", err)
	}

	return pool
}

func TestUserRepository_CRUD(t *testing.T) {
	db := getTestDBPool(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	// 1. Test CreateUser
	testID := uuid.NewString()
	name := "Budi Santoso"
	user := &model.User{
		ID:    testID,
		Email: "budi.test@" + testID[:8] + ".com",
		Name:  &name,
		Role:  "user",
	}

	err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", user.Role)
	}

	// 2. Test GetUserByID
	fetchedUser, err := repo.GetUserByID(ctx, testID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if fetchedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, fetchedUser.Email)
	}

	// 3. Test UpdateUser
	updatedName := "Budi Santoso Updated"
	fetchedUser.Name = &updatedName
	err = repo.UpdateUser(ctx, fetchedUser)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	fetchedUser2, _ := repo.GetUserByID(ctx, testID)
	if *fetchedUser2.Name != updatedName {
		t.Errorf("Expected name %s, got %s", updatedName, *fetchedUser2.Name)
	}

	// 4. Test UpsertUserPreferences
	restrictions := json.RawMessage(`["udang", "kacang"]`)
	pref := &model.UserPreference{
		UserID:        testID,
		Goal:          "hemat",
		BudgetAmount:  50000,
		BudgetPeriod:  "harian",
		HouseholdSize: 2,
		Restrictions:  restrictions,
	}

	err = repo.UpsertUserPreferences(ctx, pref)
	if err != nil {
		t.Fatalf("UpsertUserPreferences failed: %v", err)
	}

	if pref.ID == 0 {
		t.Errorf("Expected preference ID > 0, got 0")
	}

	// 5. Test GetUserPreferencesByUserID
	fetchedPref, err := repo.GetUserPreferencesByUserID(ctx, testID)
	if err != nil {
		t.Fatalf("GetUserPreferencesByUserID failed: %v", err)
	}

	if fetchedPref.Goal != "hemat" {
		t.Errorf("Expected goal 'hemat', got '%s'", fetchedPref.Goal)
	}
	if fetchedPref.BudgetAmount != 50000 {
		t.Errorf("Expected budget 50000, got %d", fetchedPref.BudgetAmount)
	}
}
