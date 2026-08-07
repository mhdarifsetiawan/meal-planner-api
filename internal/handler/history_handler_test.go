package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"

	"github.com/gofiber/fiber/v2"
)

type MockHistoryRepository struct {
	mockItems []model.HistoryItem
	mockTotal int
	mockErr   error
}

func (m *MockHistoryRepository) GetHistoryByUserID(ctx context.Context, userID string, limit int, offset int) ([]model.HistoryItem, int, error) {
	if m.mockErr != nil {
		return nil, 0, m.mockErr
	}
	return m.mockItems, m.mockTotal, nil
}

func setupHistoryTestApp(repo *MockHistoryRepository) *fiber.App {
	app := fiber.New()
	h := NewHistoryHandler(repo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Get("/api/v1/history", h.HandleGetHistory)
	return app
}

func TestHandleGetHistory_Unauthorized(t *testing.T) {
	mockRepo := &MockHistoryRepository{}
	app := setupHistoryTestApp(mockRepo)

	req := httptest.NewRequest("GET", "/api/v1/history", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleGetHistory_Success(t *testing.T) {
	shopListID := 1
	mockRepo := &MockHistoryRepository{
		mockItems: []model.HistoryItem{
			{
				MealSelectionID:     1,
				ShoppingListID:      &shopListID,
				RecipeID:            1,
				RecipeName:          "Nasi Goreng Spesial",
				Description:         "Enak & Praktis",
				SelectedDate:        time.Now(),
				TotalEstimatedPrice: 25000,
				CreatedAt:           time.Now(),
			},
		},
		mockTotal: 1,
	}
	app := setupHistoryTestApp(mockRepo)

	req := httptest.NewRequest("GET", "/api/v1/history?limit=10&offset=0", nil)
	req.Header.Set("Test-User-ID", "user-uuid-100")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}
