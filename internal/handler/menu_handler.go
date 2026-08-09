package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/price"
	"meal-planner-api/internal/repository"
	"meal-planner-api/internal/subscription"

	"github.com/gofiber/fiber/v2"
)

const DefaultFreeDailyLimit = 3

type MenuHandler struct {
	aiProvider    ai.AIProvider
	priceProvider price.PriceProvider
	userRepo      repository.UserRepository
	subRepo       repository.SubscriptionRepository
	menuRepo      repository.MenuRepository
	rateLimiter   subscription.RateLimiter
}

func NewMenuHandler(
	aiProvider ai.AIProvider,
	priceProvider price.PriceProvider,
	userRepo repository.UserRepository,
	subRepo repository.SubscriptionRepository,
	rateLimiter subscription.RateLimiter,
	menuRepo ...repository.MenuRepository,
) *MenuHandler {
	var mRepo repository.MenuRepository
	if len(menuRepo) > 0 {
		mRepo = menuRepo[0]
	}
	return &MenuHandler{
		aiProvider:    aiProvider,
		priceProvider: priceProvider,
		userRepo:      userRepo,
		subRepo:       subRepo,
		rateLimiter:   rateLimiter,
		menuRepo:      mRepo,
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

	// 1. Dynamic Rate limiting check based on user subscription plan features
	dailyLimit := DefaultFreeDailyLimit
	if h.subRepo != nil {
		if limit, lErr := h.subRepo.GetUserDailyLimit(ctx, userID); lErr == nil && limit > 0 {
			dailyLimit = limit
		}
	}

	allowed, currentCount, err := h.rateLimiter.Allow(ctx, userID, dailyLimit)
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
				"message": fmt.Sprintf("Rate limit exceeded: your current plan limit is %d menu generations per day. Please upgrade to Premium.", dailyLimit),
				"code":    "RATE_LIMIT_EXCEEDED",
				"current": currentCount,
				"max":     dailyLimit,
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

	// Fetch user record for city_id if present
	var cityID *int
	if user, uErr := h.userRepo.GetUserByID(ctx, userID); uErr == nil && user != nil {
		cityID = user.CityID
	}

	// 3. Parse restrictions JSON
	var restrictions []string
	if len(pref.Restrictions) > 0 {
		_ = json.Unmarshal(pref.Restrictions, &restrictions)
	}

	// 3b. Fetch previously generated recipe names for anti-duplication
	var excludedRecipes []string
	if h.menuRepo != nil {
		if prevGens, _, gErr := h.menuRepo.GetMenuGenerationsHistory(ctx, userID, 5, 0); gErr == nil {
			recipeMap := make(map[string]bool)
			for _, g := range prevGens {
				if optsSlice, ok := g.Options.([]interface{}); ok {
					for _, optItem := range optsSlice {
						if optMap, ok := optItem.(map[string]interface{}); ok {
							if rName, ok := optMap["recipe_name"].(string); ok && rName != "" {
								if !recipeMap[rName] {
									recipeMap[rName] = true
									excludedRecipes = append(excludedRecipes, rName)
								}
							}
						}
					}
				}
			}
		}
	}

	// 4. Construct AI Params
	params := ai.MenuGenerateParams{
		Goal:           pref.Goal,
		BudgetAmount:   pref.BudgetAmount,
		BudgetPeriod:   pref.BudgetPeriod,
		HouseholdSize:  pref.HouseholdSize,
		Restrictions:   restrictions,
		ExcludeRecipes: excludedRecipes,
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

	// 6. Enrich ingredient prices using PriceProvider if available
	if h.priceProvider != nil && menuOpts != nil {
		for i := range menuOpts.Options {
			opt := &menuOpts.Options[i]
			totalPrice := 0
			for j := range opt.Ingredients {
				ing := &opt.Ingredients[j]
				// If AI didn't provide ingredient estimated price, fallback to PriceProvider catalog
				if ing.EstimatedPrice <= 0 {
					priceRes, pErr := h.priceProvider.GetIngredientPrice(ctx, ing.Name, cityID)
					if pErr == nil && priceRes != nil && priceRes.Price > 0 {
						ing.EstimatedPrice = priceRes.Price
						ing.PriceSource = string(priceRes.Source)
					}
				}
				if ing.PriceSource == "" {
					ing.PriceSource = string(price.SourceAIEstimate)
				}
				totalPrice += ing.EstimatedPrice
			}
			if totalPrice > 0 && opt.EstimatedTotalPrice <= 0 {
				opt.EstimatedTotalPrice = totalPrice
			}
		}
	}

	// 7. Increment generation count upon success
	_ = h.rateLimiter.Increment(ctx, userID)

	// 8. Persist generated options in user_menu_generations for session retention & history
	if h.menuRepo != nil && menuOpts != nil && len(menuOpts.Options) > 0 {
		_ = h.menuRepo.SaveMenuGeneration(ctx, userID, menuOpts.Options)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  menuOpts,
		"error": nil,
	})
}

func (h *MenuHandler) HandleGetLatestMenu(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	if h.menuRepo == nil {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"data": fiber.Map{
				"options": []interface{}{},
			},
			"error": nil,
		})
	}

	gen, err := h.menuRepo.GetLatestMenuGenerationToday(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch latest menu generation: " + err.Error(),
			},
		})
	}

	if gen == nil {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"data": fiber.Map{
				"options": []interface{}{},
			},
			"error": nil,
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"id":              gen.ID,
			"options":         gen.Options,
			"generation_date": gen.GenerationDate,
			"created_at":      gen.CreatedAt,
		},
		"error": nil,
	})
}

func (h *MenuHandler) HandleGetGenerationsHistory(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	if h.menuRepo == nil {
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"data": fiber.Map{
				"generations": []interface{}{},
				"total":       0,
			},
			"error": nil,
		})
	}

	limit := 20
	offset := 0

	generations, total, err := h.menuRepo.GetMenuGenerationsHistory(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch menu generations history: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"generations": generations,
			"total":       total,
		},
		"error": nil,
	})
}

