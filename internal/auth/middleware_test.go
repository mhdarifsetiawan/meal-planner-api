package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "super-secret-jwt-key-for-unit-tests-12345"

func generateTestToken(secret string, userID string, email string, role string, expired bool) (string, error) {
	exp := time.Now().Add(1 * time.Hour)
	if expired {
		exp = time.Now().Add(-1 * time.Hour)
	}

	claims := SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: email,
		Role:  role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func setupTestApp() *fiber.App {
	app := fiber.New()

	// Protected endpoint
	app.Get("/protected", RequireAuth(testSecret), func(c *fiber.Ctx) error {
		userID, err := GetUserID(c)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		email := GetUserEmail(c)
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"user_id": userID,
				"email":   email,
			},
			"error": nil,
		})
	})

	// Admin endpoint
	app.Get("/admin", RequireAuth(testSecret), RequireAdmin(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"data":  "welcome admin",
			"error": nil,
		})
	})

	return app
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_InvalidFormat(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic invalidtokenformat")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_InvalidSecret(t *testing.T) {
	app := setupTestApp()

	token, _ := generateTestToken("wrong-secret", "user-uuid-123", "user@test.com", "user", false)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	app := setupTestApp()

	token, _ := generateTestToken(testSecret, "user-uuid-123", "user@test.com", "user", true)
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_ValidTokenSuccess(t *testing.T) {
	app := setupTestApp()

	expectedUUID := "123e4567-e89b-12d3-a456-426614174000"
	expectedEmail := "user@masakapa.com"
	token, err := generateTestToken(testSecret, expectedUUID, expectedEmail, "user", false)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
		} `json:"data"`
		Error interface{} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if res.Data.UserID != expectedUUID {
		t.Errorf("Expected user_id %s, got %s", expectedUUID, res.Data.UserID)
	}
	if res.Data.Email != expectedEmail {
		t.Errorf("Expected email %s, got %s", expectedEmail, res.Data.Email)
	}
}

func TestRequireAdmin_ForbiddenForNormalUser(t *testing.T) {
	app := setupTestApp()

	token, _ := generateTestToken(testSecret, "user-uuid-123", "user@test.com", "user", false)
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestRequireAdmin_AllowedForAdminUser(t *testing.T) {
	app := setupTestApp()

	token, _ := generateTestToken(testSecret, "admin-uuid-999", "admin@masakapa.com", "admin", false)
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
