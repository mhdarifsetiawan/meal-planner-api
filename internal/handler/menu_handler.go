package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/repository"
	"meal-planner-api/internal/subscription"

	"github.com/gofiber/fiber/v2"
)

const DefaultFreeDailyLimit = 3

type MenuHandler struct {
	aiProvider  ai.AIProvider
	userRepo    repository.UserRepository
	rateLimiter subscription.RateLimiter
}

func NewMenuHandler(aiProvider ai.AIProvider, userRepo repository.UserRepository, rateLimiter subscription.RateLimiter) *MenuHandler {
	return &MenuHandler{
		aiProvider:  aiProvider,
		userRepo:    userRepo,
		rateLimiter: rateLimiter,
	}
}

func (h *MenuHandler) HandleGenerateMenu(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	ctx := c.Context()

	// 1. Rate limiting check
	allowed, currentCount, err := h.rateLimiter.Allow(ctx, userID, DefaultFreeDailyLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Rate limit check error: " + err.Error(),
			},
		})
	}

	if !allowed {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Rate limit exceeded: free tier limit is 3 menu generations per day. Please upgrade to Premium.",
				"code":    "RATE_LIMIT_EXCEEDED",
				"current": currentCount,
				"max":     DefaultFreeDailyLimit,
			},
		})
	}

	// 2. Fetch User Preferences
	pref, err := h.userRepo.GetUserPreferencesByUserID(ctx, userID)
	if err != nil || pref == nil {
		if errors.Is(err, repository.ErrUserPreferenceNotFound) || pref == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "User preferences not found. Please complete onboarding first.",
					"code":    "ONBOARDING_REQUIRED",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch user preferences: " + err.Error(),
			},
		})
	}

	// 3. Parse restrictions JSON
	var restrictions []string
	if len(pref.Restrictions) > 0 {
		_ = json.Unmarshal(pref.Restrictions, &restrictions)
	}

	// 4. Construct AI Params
	params := ai.MenuGenerateParams{
		Goal:          pref.Goal,
		BudgetAmount:  pref.BudgetAmount,
		BudgetPeriod:  pref.BudgetPeriod,
		HouseholdSize: pref.HouseholdSize,
		Restrictions:  restrictions,
	}

	// 5. Generate Menu via AI Provider
	menuOpts, err := h.aiProvider.GenerateMenu(ctx, params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to generate menu options: " + err.Error(),
			},
		})
	}

	// 6. Increment generation count upon success
	_ = h.rateLimiter.Increment(ctx, userID)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  menuOpts,
		"error": nil,
	})
}
