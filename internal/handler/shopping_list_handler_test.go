package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type MockShoppingListRepository struct {
	mockDetail *model.ShoppingListDetail
	mockItem   *model.ShoppingItem
	mockErr    error
}

func (m *MockShoppingListRepository) GetShoppingListByID(ctx context.Context, id int, userID string) (*model.ShoppingListDetail, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if m.mockDetail != nil {
		return m.mockDetail, nil
	}
	return nil, repository.ErrShoppingListNotFound
}

func (m *MockShoppingListRepository) UpdateShoppingListItemChecklist(ctx context.Context, id int, userID string, ingredientName string, isChecked bool) (*model.ShoppingItem, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if m.mockItem != nil {
		m.mockItem.IsChecked = isChecked
		return m.mockItem, nil
	}
	return &model.ShoppingItem{
		IngredientName: ingredientName,
		Quantity:       "1",
		Unit:           "papan",
		EstimatedPrice: 5000,
		IsChecked:      isChecked,
	}, nil
}

func (m *MockShoppingListRepository) UpdateShoppingListItemPrice(ctx context.Context, id int, userID string, ingredientName string, newPrice int, submitToCommunity bool) (*model.ShoppingItem, int, bool, error) {
	if m.mockErr != nil {
		return nil, 0, false, m.mockErr
	}
	if m.mockItem != nil {
		m.mockItem.EstimatedPrice = newPrice
		return m.mockItem, newPrice, submitToCommunity, nil
	}
	return &model.ShoppingItem{
		IngredientName: ingredientName,
		Quantity:       "1",
		Unit:           "papan",
		EstimatedPrice: newPrice,
		IsChecked:      false,
	}, newPrice, submitToCommunity, nil
}

func setupShoppingListTestApp(repo *MockShoppingListRepository) *fiber.App {
	app := fiber.New()
	h := NewShoppingListHandler(repo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Get("/api/v1/shopping-list/:id", h.HandleGetShoppingList)
	app.Patch("/api/v1/shopping-list/:id/item", h.HandleUpdateShoppingListItem)
	return app
}

func TestHandleGetShoppingList_Unauthorized(t *testing.T) {
	mockRepo := &MockShoppingListRepository{}
	app := setupShoppingListTestApp(mockRepo)

	req := httptest.NewRequest("GET", "/api/v1/shopping-list/1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleGetShoppingList_NotFound(t *testing.T) {
	mockRepo := &MockShoppingListRepository{mockErr: repository.ErrShoppingListNotFound}
	app := setupShoppingListTestApp(mockRepo)

	req := httptest.NewRequest("GET", "/api/v1/shopping-list/999", nil)
	req.Header.Set("Test-User-ID", "user-uuid-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleGetShoppingList_Success(t *testing.T) {
	mockRepo := &MockShoppingListRepository{
		mockDetail: &model.ShoppingListDetail{
			ID:              1,
			MealSelectionID: 1,
			RecipeName:      "Tahu Tempe Bacem",
			Items: []model.ShoppingItem{
				{IngredientName: "Tahu Putih", Quantity: "1", Unit: "papan", EstimatedPrice: 5000, IsChecked: false},
			},
			TotalEstimatedPrice: 5000,
			CreatedAt:           time.Now(),
		},
	}
	app := setupShoppingListTestApp(mockRepo)

	req := httptest.NewRequest("GET", "/api/v1/shopping-list/1", nil)
	req.Header.Set("Test-User-ID", "user-uuid-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestHandleUpdateShoppingListItem_Success(t *testing.T) {
	mockRepo := &MockShoppingListRepository{
		mockItem: &model.ShoppingItem{
			IngredientName: "Tahu Putih",
			Quantity:       "1",
			Unit:           "papan",
			EstimatedPrice: 5000,
			IsChecked:      true,
		},
	}
	app := setupShoppingListTestApp(mockRepo)

	body := UpdateChecklistRequest{
		IngredientName: "Tahu Putih",
		IsChecked:      true,
	}
	jsonBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PATCH", "/api/v1/shopping-list/1/item", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "user-uuid-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}
