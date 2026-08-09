package handler

import (
	"net/http"
	"strconv"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type HistoryHandler struct {
	repo repository.HistoryRepository
}

func NewHistoryHandler(repo repository.HistoryRepository) *HistoryHandler {
	return &HistoryHandler{repo: repo}
}

func (h *HistoryHandler) HandleGetHistory(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	limitStr := c.Query("limit", "20")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	items, total, err := h.repo.GetHistoryByUserID(c.Context(), userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to get user menu history: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"history": items,
			"total":   total,
		},
		"error": nil,
	})
}

func (h *HistoryHandler) HandleDeleteHistory(c *fiber.Ctx) error {
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
	historyID, err := strconv.Atoi(idStr)
	if err != nil || historyID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid history item ID",
			},
		})
	}

	if err := h.repo.DeleteHistoryItem(c.Context(), userID, historyID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": fiber.Map{
			"message": "History item deleted successfully",
		},
		"error": nil,
	})
}
