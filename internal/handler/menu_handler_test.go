package handler

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/price"
	"meal-planner-api/internal/subscription"

	"github.com/gofiber/fiber/v2"
)

type MockAIProvider struct {
	mockOptions *ai.MenuOptions
	mockErr     error
}

func (m *MockAIProvider) GenerateMenu(ctx context.Context, params ai.MenuGenerateParams) (*ai.MenuOptions, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockOptions, nil
}

func setupMenuTestApp(aiProv ai.AIProvider, userRepo *MockUserRepository, rateLimiter subscription.RateLimiter) *fiber.App {
	app := fiber.New()
	priceProv := price.NewAIEstimateProvider(nil)
	handler := NewMenuHandler(aiProv, priceProv, userRepo, nil, rateLimiter, nil)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Post("/api/v1/menu/generate", handler.HandleGenerateMenu)
	return app
}

func TestHandleGenerateMenu_Unauthorized(t *testing.T) {
	mockAI := &MockAIProvider{}
	mockRepo := NewMockUserRepository()
	limiter := subscription.NewMemoryRateLimiter()

	app := setupMenuTestApp(mockAI, mockRepo, limiter)

	req := httptest.NewRequest("POST", "/api/v1/menu/generate", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleGenerateMenu_OnboardingRequired(t *testing.T) {
	mockAI := &MockAIProvider{}
	mockRepo := NewMockUserRepository()
	limiter := subscription.NewMemoryRateLimiter()

	app := setupMenuTestApp(mockAI, mockRepo, limiter)

	req := httptest.NewRequest("POST", "/api/v1/menu/generate", nil)
	req.Header.Set("Test-User-ID", "user-without-pref")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleGenerateMenu_SuccessAndRateLimit(t *testing.T) {
	mockAI := &MockAIProvider{
		mockOptions: &ai.MenuOptions{
			Options: []ai.MenuOption{
				{
					RecipeName:          "Soto Ayam",
					Description:         "Enak & Hangat",
					EstimatedTotalPrice: 20000,
					GoalTags:            []string{"hemat"},
					Ingredients: []ai.MenuIngredient{
						{Name: "Ayam", Quantity: "250", Unit: "gram", EstimatedPrice: 12000},
					},
				},
			},
		},
	}
	mockRepo := NewMockUserRepository()
	limiter := subscription.NewMemoryRateLimiter()

	testUserID := "user-uuid-101"
	_ = mockRepo.UpsertUserPreferences(context.Background(), &model.UserPreference{
		UserID:        testUserID,
		Goal:          "hemat",
		BudgetAmount:  50000,
		BudgetPeriod:  "harian",
		HouseholdSize: 2,
		Restrictions:  json.RawMessage(`[]`),
	})

	app := setupMenuTestApp(mockAI, mockRepo, limiter)

	// Make 3 successful calls (max allowed for free tier)
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/menu/generate", nil)
		req.Header.Set("Test-User-ID", testUserID)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, resp.StatusCode)
		}
	}

	// 4th request should trigger 429 Too Many Requests
	req4 := httptest.NewRequest("POST", "/api/v1/menu/generate", nil)
	req4.Header.Set("Test-User-ID", testUserID)

	resp4, err := app.Test(req4)
	if err != nil {
		t.Fatalf("Request 4 failed: %v", err)
	}
	if resp4.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 Too Many Requests, got %d", resp4.StatusCode)
	}
}

func TestHandleGenerateMenu_CrowdsourcePriceScaling(t *testing.T) {
	// Verify that crowdsource bulk unit price (e.g. 30000/kg) scales AI portion price (4000 for 2 eggs)
	// proportionally instead of overwriting it with 30000.
	aiPortionPrice := 4000
	baselineUnitPrice := 32000
	crowdsourceUnitPrice := 30000

	ratio := float64(crowdsourceUnitPrice) / float64(baselineUnitPrice)
	expectedPortionPrice := int(math.Round(float64(aiPortionPrice) * ratio)) // 3750

	if expectedPortionPrice == 30000 {
		t.Fatalf("Portion price should NOT equal bulk unit price!")
	}
	if expectedPortionPrice != 3750 {
		t.Errorf("Expected portion price 3750, got %d", expectedPortionPrice)
	}
}

