package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"

	"github.com/gofiber/fiber/v2"
)

type MockCreditRepository struct {
	balances map[string]int
}

func NewMockCreditRepository() *MockCreditRepository {
	return &MockCreditRepository{
		balances: make(map[string]int),
	}
}

func (m *MockCreditRepository) GetUserCreditSummary(ctx context.Context, userID string) (*model.UserCreditSummary, error) {
	bal := m.balances[userID]
	return &model.UserCreditSummary{
		Balance:      bal,
		Transactions: []model.CreditTransaction{},
	}, nil
}

func (m *MockCreditRepository) AddCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error {
	m.balances[userID] += amount
	return nil
}

func (m *MockCreditRepository) DeductCredit(ctx context.Context, userID string, amount int, txType string, refID *int) error {
	m.balances[userID] -= amount
	return nil
}

func setupCreditTestApp(repo *MockCreditRepository) *fiber.App {
	app := fiber.New()
	h := NewCreditHandler(repo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Get("/api/v1/credits/me", h.HandleGetMyCredits)
	return app
}

func TestCreditHandler_GetMyCredits(t *testing.T) {
	repo := NewMockCreditRepository()
	app := setupCreditTestApp(repo)

	// Unauthorized
	reqUnauth := httptest.NewRequest("GET", "/api/v1/credits/me", nil)
	respUnauth, err := app.Test(reqUnauth)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respUnauth.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", respUnauth.StatusCode)
	}

	// Authorized
	reqAuth := httptest.NewRequest("GET", "/api/v1/credits/me", nil)
	reqAuth.Header.Set("Test-User-ID", "credit-user-123")
	respAuth, err := app.Test(reqAuth)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respAuth.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", respAuth.StatusCode)
	}
}
