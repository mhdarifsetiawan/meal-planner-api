package handler

import (
	"net/http"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type CreditHandler struct {
	repo repository.CreditRepository
}

func NewCreditHandler(repo repository.CreditRepository) *CreditHandler {
	return &CreditHandler{repo: repo}
}

func (h *CreditHandler) HandleGetMyCredits(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	summary, err := h.repo.GetUserCreditSummary(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch credit summary: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  summary,
		"error": nil,
	})
}
