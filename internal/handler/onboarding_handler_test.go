package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

// MockUserRepository implements repository.UserRepository for testing
type MockUserRepository struct {
	users map[string]*model.User
	prefs map[string]*model.UserPreference
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*model.User),
		prefs: make(map[string]*model.UserPreference),
	}
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) UpsertUserPreferences(ctx context.Context, pref *model.UserPreference) error {
	pref.ID = 1
	m.prefs[pref.UserID] = pref
	return nil
}

func (m *MockUserRepository) GetUserPreferencesByUserID(ctx context.Context, userID string) (*model.UserPreference, error) {
	p, ok := m.prefs[userID]
	if !ok {
		return nil, repository.ErrUserPreferenceNotFound
	}
	return p, nil
}

func setupOnboardingTestApp(mockRepo *MockUserRepository) *fiber.App {
	app := fiber.New()

	handler := NewOnboardingHandler(mockRepo)

	// Inject fake auth context
	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
			c.Locals(auth.LocalEmailKey, userID+"@test.com")
		}
		return c.Next()
	})

	app.Post("/api/v1/onboarding", handler.HandleOnboarding)
	return app
}

func TestHandleOnboarding_Unauthorized(t *testing.T) {
	mockRepo := NewMockUserRepository()
	app := setupOnboardingTestApp(mockRepo)

	body := []byte(`{"goal":"hemat","budget_amount":50000,"budget_period":"harian","household_size":2}`)
	req := httptest.NewRequest("POST", "/api/v1/onboarding", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleOnboarding_ValidationErrors(t *testing.T) {
	mockRepo := NewMockUserRepository()
	app := setupOnboardingTestApp(mockRepo)

	tests := []struct {
		name       string
		payload    string
		expectCode int
	}{
		{
			name:       "Invalid Goal",
			payload:    `{"goal":"super_cheap","budget_amount":50000,"budget_period":"harian","household_size":2}`,
			expectCode: 400,
		},
		{
			name:       "Zero Budget",
			payload:    `{"goal":"hemat","budget_amount":0,"budget_period":"harian","household_size":2}`,
			expectCode: 400,
		},
		{
			name:       "Invalid Budget Period",
			payload:    `{"goal":"hemat","budget_amount":50000,"budget_period":"bulanan","household_size":2}`,
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/onboarding", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Test-User-ID", "user-uuid-123")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, resp.StatusCode)
			}
		})
	}
}

func TestHandleOnboarding_Success(t *testing.T) {
	mockRepo := NewMockUserRepository()
	app := setupOnboardingTestApp(mockRepo)

	payload := `{
		"goal": "sehat",
		"budget_amount": 100000,
		"budget_period": "mingguan",
		"household_size": 3,
		"restrictions": ["udang", "kacang"]
	}`

	req := httptest.NewRequest("POST", "/api/v1/onboarding", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "user-uuid-999")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var res struct {
		Data struct {
			UserID      string                `json:"user_id"`
			Preferences model.UserPreference `json:"preferences"`
		} `json:"data"`
		Error interface{} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if res.Data.UserID != "user-uuid-999" {
		t.Errorf("Expected user_id 'user-uuid-999', got '%s'", res.Data.UserID)
	}

	if res.Data.Preferences.Goal != "sehat" {
		t.Errorf("Expected goal 'sehat', got '%s'", res.Data.Preferences.Goal)
	}

	if res.Data.Preferences.BudgetAmount != 100000 {
		t.Errorf("Expected budget 100000, got %d", res.Data.Preferences.BudgetAmount)
	}
}
