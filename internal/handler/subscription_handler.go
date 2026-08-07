package handler

import (
	"net/http"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/payment"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type SubscribeRequest struct {
	PlanName   string `json:"plan_name"`   // "free" | "premium"
	CouponCode string `json:"coupon_code"` // optional
}

type SubscriptionHandler struct {
	subRepo         repository.SubscriptionRepository
	paymentProvider payment.PaymentProvider
}

func NewSubscriptionHandler(subRepo repository.SubscriptionRepository, paymentProvider payment.PaymentProvider) *SubscriptionHandler {
	return &SubscriptionHandler{
		subRepo:         subRepo,
		paymentProvider: paymentProvider,
	}
}

func (h *SubscriptionHandler) HandleSubscribe(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	var req SubscribeRequest
	if err := c.BodyParser(&req); err != nil && len(c.Body()) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	if req.PlanName == "" {
		req.PlanName = "premium"
	}

	ctx := c.Context()

	// 1. Get plan details
	plan, err := h.subRepo.GetSubscriptionPlanByName(ctx, req.PlanName)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid subscription plan: " + req.PlanName,
			},
		})
	}

	// 2. Process Payment via PaymentProvider
	payReq := payment.PaymentRequest{
		UserID:        userID,
		PlanID:        plan.ID,
		Amount:        plan.Price,
		PaymentMethod: "dummy",
	}

	payResp, err := h.paymentProvider.ProcessPayment(ctx, payReq)
	if err != nil || payResp.Status != payment.StatusSuccess {
		errMsg := "Payment processing failed"
		if err != nil {
			errMsg += ": " + err.Error()
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": errMsg,
			},
		})
	}

	// 3. Persist subscription transaction in DB
	result, err := h.subRepo.CreateUserSubscriptionTx(ctx, userID, plan, nil, payResp.Amount, "dummy", payResp.GatewayRef)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to create subscription record: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  result,
		"error": nil,
	})
}
