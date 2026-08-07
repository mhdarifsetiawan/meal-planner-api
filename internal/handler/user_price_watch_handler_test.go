package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/auth"

	"github.com/gofiber/fiber/v2"
)

func setupUserPriceWatchTestApp(pwRepo *MockPriceWatchRepository) *fiber.App {
	app := fiber.New()
	h := NewUserPriceWatchHandler(pwRepo)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Get("/api/v1/price-watch/campaigns/active", h.HandleGetActiveCampaigns)
	app.Get("/api/v1/price-watch/submissions/me", h.HandleGetUserSubmissions)
	return app
}

func TestUserPriceWatchHandler_GetActiveCampaigns(t *testing.T) {
	pwRepo := NewMockPriceWatchRepository()
	app := setupUserPriceWatchTestApp(pwRepo)

	req := httptest.NewRequest("GET", "/api/v1/price-watch/campaigns/active", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestUserPriceWatchHandler_GetUserSubmissions(t *testing.T) {
	pwRepo := NewMockPriceWatchRepository()
	app := setupUserPriceWatchTestApp(pwRepo)

	// Unauthorized
	reqUnauth := httptest.NewRequest("GET", "/api/v1/price-watch/submissions/me", nil)
	respUnauth, err := app.Test(reqUnauth)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respUnauth.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", respUnauth.StatusCode)
	}

	// Authorized
	reqAuth := httptest.NewRequest("GET", "/api/v1/price-watch/submissions/me", nil)
	reqAuth.Header.Set("Test-User-ID", "user-uuid-123")
	respAuth, err := app.Test(reqAuth)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respAuth.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", respAuth.StatusCode)
	}
}
