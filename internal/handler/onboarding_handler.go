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
	validGoals         = []string{"hemat", "sehat", "diet", "bebas", "variatif", "praktis"}
	validBudgetPeriods = []string{"harian", "mingguan", "bulanan", "daily", "weekly", "monthly"}
)

type OnboardingRequest struct {
	Goal                string   `json:"goal"`
	BudgetAmount        int      `json:"budget_amount"`
	DailyBudget         int      `json:"daily_budget"`
	MonthlyBudget       int      `json:"monthly_budget"`
	BudgetPeriod        string   `json:"budget_period"`
	HouseholdSize       int      `json:"household_size"`
	FamilyMembersCount  int      `json:"family_members_count"`
	Restrictions        []string `json:"restrictions"`
	DietaryRestrictions []string `json:"dietary_restrictions"`
	CityID              *int     `json:"city_id,omitempty"`
}

func (req *OnboardingRequest) Validate() error {
	if req.BudgetAmount <= 0 {
		if req.DailyBudget > 0 {
			req.BudgetAmount = req.DailyBudget
		} else if req.MonthlyBudget > 0 {
			req.BudgetAmount = req.MonthlyBudget
		}
	}
	if req.HouseholdSize <= 0 && req.FamilyMembersCount > 0 {
		req.HouseholdSize = req.FamilyMembersCount
	}
	if len(req.Restrictions) == 0 && len(req.DietaryRestrictions) > 0 {
		req.Restrictions = req.DietaryRestrictions
	}
	if req.BudgetPeriod == "daily" {
		req.BudgetPeriod = "harian"
	} else if req.BudgetPeriod == "weekly" {
		req.BudgetPeriod = "mingguan"
	} else if req.BudgetPeriod == "monthly" {
		req.BudgetPeriod = "bulanan"
	}

	if !slices.Contains(validGoals, req.Goal) {
		return fmt.Errorf("goal must be one of [hemat, sehat, diet, bebas, variatif, praktis]")
	}
	if req.BudgetAmount <= 0 {
		return fmt.Errorf("budget_amount must be greater than 0")
	}
	if !slices.Contains(validBudgetPeriods, req.BudgetPeriod) {
		return fmt.Errorf("budget_period must be one of [harian, mingguan, bulanan]")
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
