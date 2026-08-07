package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"

	"github.com/gofiber/fiber/v2"
)

func setupSubmissionTestApp(pwRepo *MockPriceWatchRepository, userRepo *MockUserRepository) *fiber.App {
	app := fiber.New()
	h := NewPriceWatchSubmissionHandler(pwRepo, userRepo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Post("/api/v1/price-watch/submissions", h.HandleSubmitPrice)
	return app
}

func TestHandleSubmitPrice_Unauthorized(t *testing.T) {
	pwRepo := NewMockPriceWatchRepository()
	userRepo := NewMockUserRepository()
	app := setupSubmissionTestApp(pwRepo, userRepo)

	req := httptest.NewRequest("POST", "/api/v1/price-watch/submissions", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleSubmitPrice_MissingCityID(t *testing.T) {
	pwRepo := NewMockPriceWatchRepository()
	userRepo := NewMockUserRepository()
	app := setupSubmissionTestApp(pwRepo, userRepo)

	body := CreateSubmissionRequest{
		WatchItemID:    1,
		SubmittedPrice: 15000,
	}
	jsonBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/price-watch/submissions", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "user-without-city")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestHandleSubmitPrice_Success(t *testing.T) {
	pwRepo := NewMockPriceWatchRepository()
	userRepo := NewMockUserRepository()
	_ = pwRepo.CreateItem(nil, &model.PriceWatchItem{
		CampaignID:     1,
		IngredientName: "Beras SPHP",
		Unit:           "kg",
	})

	app := setupSubmissionTestApp(pwRepo, userRepo)

	cityID := 1
	body := CreateSubmissionRequest{
		WatchItemID:    1,
		SubmittedPrice: 14500,
		CityID:         &cityID,
	}
	jsonBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/price-watch/submissions", bytes.NewReader(jsonBytes))
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
