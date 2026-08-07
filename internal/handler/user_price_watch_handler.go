package handler

import (
	"net/http"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type UserPriceWatchHandler struct {
	pwRepo repository.PriceWatchRepository
}

func NewUserPriceWatchHandler(pwRepo repository.PriceWatchRepository) *UserPriceWatchHandler {
	return &UserPriceWatchHandler{pwRepo: pwRepo}
}

func (h *UserPriceWatchHandler) HandleGetActiveCampaigns(c *fiber.Ctx) error {
	campaigns, err := h.pwRepo.GetActiveCampaignsWithItems(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch active campaigns: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  campaigns,
		"error": nil,
	})
}

func (h *UserPriceWatchHandler) HandleGetUserSubmissions(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	submissions, err := h.pwRepo.GetUserSubmissions(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch user submissions: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  submissions,
		"error": nil,
	})
}
