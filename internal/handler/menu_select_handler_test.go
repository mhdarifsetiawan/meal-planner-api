package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"

	"github.com/gofiber/fiber/v2"
)

type MockMenuRepository struct {
	mockResult *model.SelectMenuResult
	mockErr    error
}

func (m *MockMenuRepository) CreateSelectedMenuAndShoppingList(
	ctx context.Context,
	userID string,
	recipe *model.Recipe,
	ingredients []model.RecipeIngredient,
) (*model.SelectMenuResult, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if m.mockResult != nil {
		return m.mockResult, nil
	}
	return &model.SelectMenuResult{
		UserMenuID:          1,
		ShoppingListID:      1,
		RecipeName:          recipe.Name,
		TotalEstimatedPrice: recipe.EstimatedTotalPrice,
		ItemsCount:          len(ingredients),
		CreatedAt:           time.Now(),
	}, nil
}

func setupMenuSelectTestApp(menuRepo *MockMenuRepository) *fiber.App {
	app := fiber.New()
	handler := NewMenuSelectHandler(menuRepo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Post("/api/v1/menu/select", handler.HandleSelectMenu)
	return app
}

func TestHandleSelectMenu_Unauthorized(t *testing.T) {
	mockRepo := &MockMenuRepository{}
	app := setupMenuSelectTestApp(mockRepo)

	req := httptest.NewRequest("POST", "/api/v1/menu/select", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleSelectMenu_ValidationErrors(t *testing.T) {
	mockRepo := &MockMenuRepository{}
	app := setupMenuSelectTestApp(mockRepo)

	tests := []struct {
		name string
		body SelectMenuRequest
	}{
		{
			name: "Missing Recipe Name",
			body: SelectMenuRequest{
				RecipeName: "",
				Ingredients: []ai.MenuIngredient{
					{Name: "Tahu", Quantity: "1", Unit: "papan", EstimatedPrice: 5000},
				},
			},
		},
		{
			name: "Empty Ingredients",
			body: SelectMenuRequest{
				RecipeName:  "Tahu Tempe Bacem",
				Ingredients: []ai.MenuIngredient{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/menu/select", bytes.NewReader(jsonBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Test-User-ID", "user-uuid-99")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected status 400 Bad Request, got %d", resp.StatusCode)
			}
		})
	}
}

func TestHandleSelectMenu_Success(t *testing.T) {
	mockRepo := &MockMenuRepository{}
	app := setupMenuSelectTestApp(mockRepo)

	body := SelectMenuRequest{
		RecipeName:          "Tahu Tempe Bacem",
		Description:         "Enak & Hemat",
		EstimatedTotalPrice: 11000,
		GoalTags:            []string{"hemat"},
		Ingredients: []ai.MenuIngredient{
			{Name: "Tahu Putih", Quantity: "1", Unit: "papan", EstimatedPrice: 5000},
			{Name: "Tempe", Quantity: "1", Unit: "papan", EstimatedPrice: 6000},
		},
	}

	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/menu/select", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "user-uuid-99")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", resp.StatusCode)
	}
}
