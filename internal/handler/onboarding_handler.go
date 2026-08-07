package handler

import (
	"encoding/json"
	"fmt"
	"slices"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

var (
	validGoals         = []string{"hemat", "sehat", "diet", "bebas"}
	validBudgetPeriods = []string{"harian", "mingguan"}
)

type OnboardingRequest struct {
	Goal          string   `json:"goal"`
	BudgetAmount  int      `json:"budget_amount"`
	BudgetPeriod  string   `json:"budget_period"`
	HouseholdSize int      `json:"household_size"`
	Restrictions  []string `json:"restrictions"`
	CityID        *int     `json:"city_id,omitempty"`
}

func (req *OnboardingRequest) Validate() error {
	if !slices.Contains(validGoals, req.Goal) {
		return fmt.Errorf("goal must be one of [hemat, sehat, diet, bebas]")
	}
	if req.BudgetAmount <= 0 {
		return fmt.Errorf("budget_amount must be greater than 0")
	}
	if !slices.Contains(validBudgetPeriods, req.BudgetPeriod) {
		return fmt.Errorf("budget_period must be one of [harian, mingguan]")
	}
	if req.HouseholdSize < 1 {
		req.HouseholdSize = 1
	}
	if req.Restrictions == nil {
		req.Restrictions = []string{}
	}
	return nil
}

type OnboardingHandler struct {
	userRepo repository.UserRepository
}

func NewOnboardingHandler(userRepo repository.UserRepository) *OnboardingHandler {
	return &OnboardingHandler{userRepo: userRepo}
}

// HandleOnboarding processes POST /onboarding requests.
func (h *OnboardingHandler) HandleOnboarding(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: user_id missing from context",
			},
		})
	}
	email := auth.GetUserEmail(c)

	var req OnboardingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request payload: " + err.Error(),
			},
		})
	}

	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Validation failed: " + err.Error(),
			},
		})
	}

	ctx := c.Context()

	// 1. Ensure User record exists
	user := &model.User{
		ID:     userID,
		Email:  email,
		CityID: req.CityID,
		Role:   "user",
	}
	if err := h.userRepo.CreateUser(ctx, user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to save user record: " + err.Error(),
			},
		})
	}

	// 2. Prepare JSON restrictions
	restrictionsJSON, err := json.Marshal(req.Restrictions)
	if err != nil {
		restrictionsJSON = []byte("[]")
	}

	// 3. Upsert User Preferences
	pref := &model.UserPreference{
		UserID:        userID,
		Goal:          req.Goal,
		BudgetAmount:  req.BudgetAmount,
		BudgetPeriod:  req.BudgetPeriod,
		HouseholdSize: req.HouseholdSize,
		Restrictions:  restrictionsJSON,
	}

	if err := h.userRepo.UpsertUserPreferences(ctx, pref); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to save user preferences: " + err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"user_id":     userID,
			"preferences": pref,
		},
		"error": nil,
	})
}
