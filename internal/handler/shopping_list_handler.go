package handler

import (
	"errors"
	"net/http"
	"strconv"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type UpdateChecklistRequest struct {
	IngredientName string `json:"ingredient_name"`
	IsChecked      bool   `json:"is_checked"`
}

type ShoppingListHandler struct {
	repo repository.ShoppingListRepository
}

func NewShoppingListHandler(repo repository.ShoppingListRepository) *ShoppingListHandler {
	return &ShoppingListHandler{repo: repo}
}

func (h *ShoppingListHandler) HandleGetShoppingList(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid shopping list ID parameter",
			},
		})
	}

	detail, err := h.repo.GetShoppingListByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrShoppingListNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Shopping list not found",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to get shopping list: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  detail,
		"error": nil,
	})
}

func (h *ShoppingListHandler) HandleUpdateShoppingListItem(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid shopping list ID parameter",
			},
		})
	}

	var req UpdateChecklistRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	if req.IngredientName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "ingredient_name is required",
			},
		})
	}

	updatedItem, err := h.repo.UpdateShoppingListItemChecklist(c.Context(), id, userID, req.IngredientName, req.IsChecked)
	if err != nil {
		if errors.Is(err, repository.ErrShoppingListNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Shopping list not found",
				},
			})
		}
		if errors.Is(err, repository.ErrShoppingItemNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Ingredient item not found in shopping list",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update item checklist: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"id":           id,
			"updated_item": updatedItem,
		},
		"error": nil,
	})
}

type UpdateItemPriceRequest struct {
	IngredientName    string `json:"ingredient_name"`
	RealPrice         int    `json:"real_price"`
	SubmitToCommunity bool   `json:"submit_to_community"`
}

func (h *ShoppingListHandler) HandleUpdateShoppingListItemPrice(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid shopping list ID parameter",
			},
		})
	}

	var req UpdateItemPriceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	if req.IngredientName == "" || req.RealPrice < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "ingredient_name and non-negative real_price are required",
			},
		})
	}

	updatedItem, newTotal, err := h.repo.UpdateShoppingListItemPrice(c.Context(), id, userID, req.IngredientName, req.RealPrice, req.SubmitToCommunity)
	if err != nil {
		if errors.Is(err, repository.ErrShoppingListNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Shopping list not found",
				},
			})
		}
		if errors.Is(err, repository.ErrShoppingItemNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Ingredient item not found in shopping list",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update item price: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"id":                       id,
			"updated_item":             updatedItem,
			"new_total_estimated_price": newTotal,
			"reward_credits_earned":    100,
		},
		"error": nil,
	})
}
