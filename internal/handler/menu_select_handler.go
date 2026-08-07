package handler

import (
	"net/http"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type SelectMenuRequest struct {
	RecipeName          string            `json:"recipe_name"`
	Description         string            `json:"description"`
	EstimatedTotalPrice int               `json:"estimated_total_price"`
	GoalTags            []string          `json:"goal_tags"`
	Ingredients         []ai.MenuIngredient `json:"ingredients"`
}

type MenuSelectHandler struct {
	menuRepo repository.MenuRepository
}

func NewMenuSelectHandler(menuRepo repository.MenuRepository) *MenuSelectHandler {
	return &MenuSelectHandler{menuRepo: menuRepo}
}

func (h *MenuSelectHandler) HandleSelectMenu(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	var req SelectMenuRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	if req.RecipeName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "recipe_name is required",
			},
		})
	}

	if len(req.Ingredients) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "ingredients list cannot be empty",
			},
		})
	}

	recipe := &model.Recipe{
		Name:                req.RecipeName,
		Description:         req.Description,
		EstimatedTotalPrice: req.EstimatedTotalPrice,
	}

	var ingredients []model.RecipeIngredient
	for _, ing := range req.Ingredients {
		ingredients = append(ingredients, model.RecipeIngredient{
			Name:           ing.Name,
			Quantity:       ing.Quantity,
			Unit:           ing.Unit,
			EstimatedPrice: ing.EstimatedPrice,
		})
	}

	result, err := h.menuRepo.CreateSelectedMenuAndShoppingList(c.Context(), userID, recipe, ingredients)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to save selected menu and generate shopping list: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  result,
		"error": nil,
	})
}
