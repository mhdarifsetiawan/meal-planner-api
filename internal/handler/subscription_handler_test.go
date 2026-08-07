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
	"meal-planner-api/internal/payment"

	"github.com/gofiber/fiber/v2"
)

type MockSubscriptionRepository struct {
	mockPlan *model.SubscriptionPlan
	mockErr  error
}

func (m *MockSubscriptionRepository) GetSubscriptionPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	if m.mockPlan != nil {
		return []model.SubscriptionPlan{*m.mockPlan}, nil
	}
	return []model.SubscriptionPlan{}, nil
}

func (m *MockSubscriptionRepository) GetSubscriptionPlanByName(ctx context.Context, name string) (*model.SubscriptionPlan, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	if m.mockPlan != nil {
		return m.mockPlan, nil
	}
	period := "bulanan"
	return &model.SubscriptionPlan{
		ID:            2,
		Name:          "premium",
		Price:         29000,
		BillingPeriod: &period,
	}, nil
}

func (m *MockSubscriptionRepository) CreateUserSubscriptionTx(
	ctx context.Context,
	userID string,
	plan *model.SubscriptionPlan,
	couponID *int,
	amount int,
	gateway string,
	gatewayRef string,
) (*model.UserSubscriptionResult, error) {
	now := time.Now()
	ends := now.AddDate(0, 1, 0)
	return &model.UserSubscriptionResult{
		SubscriptionID: 1,
		PlanName:       plan.Name,
		Status:         "active",
		Amount:         amount,
		PaymentGateway: gateway,
		GatewayRef:     gatewayRef,
		StartedAt:      now,
		EndsAt:         &ends,
	}, nil
}

func setupSubscriptionTestApp(subRepo *MockSubscriptionRepository, payProvider payment.PaymentProvider) *fiber.App {
	app := fiber.New()
	h := NewSubscriptionHandler(subRepo, payProvider)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	app.Post("/api/v1/subscription/subscribe", h.HandleSubscribe)
	return app
}

func TestHandleSubscribe_Unauthorized(t *testing.T) {
	subRepo := &MockSubscriptionRepository{}
	payProvider := payment.NewDummyPaymentProvider(0)
	app := setupSubscriptionTestApp(subRepo, payProvider)

	req := httptest.NewRequest("POST", "/api/v1/subscription/subscribe", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestHandleSubscribe_Success(t *testing.T) {
	subRepo := &MockSubscriptionRepository{}
	payProvider := payment.NewDummyPaymentProvider(0)
	app := setupSubscriptionTestApp(subRepo, payProvider)

	body := SubscribeRequest{PlanName: "premium"}
	jsonBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/subscription/subscribe", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "user-uuid-88")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}
