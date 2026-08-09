package handler

import (
	"errors"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type SelectAIConfigRequest struct {
	ProviderName string `json:"provider_name"`
}

type AdminAIConfigHandler struct {
	aiConfigRepo repository.AIConfigRepository
}

func NewAdminAIConfigHandler(aiConfigRepo repository.AIConfigRepository) *AdminAIConfigHandler {
	return &AdminAIConfigHandler{
		aiConfigRepo: aiConfigRepo,
	}
}

// HandleGetConfigs returns GET /api/v1/admin/ai/configs
func (h *AdminAIConfigHandler) HandleGetConfigs(c *fiber.Ctx) error {
	ctx := c.Context()
	configs, err := h.aiConfigRepo.GetAllConfigs(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch AI provider configs: " + err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"data":  configs,
		"error": nil,
	})
}

// HandleSelectConfig handles POST /api/v1/admin/ai/configs/select
func (h *AdminAIConfigHandler) HandleSelectConfig(c *fiber.Ctx) error {
	var req SelectAIConfigRequest
	if err := c.BodyParser(&req); err != nil || req.ProviderName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request payload: provider_name is required",
			},
		})
	}

	ctx := c.Context()
	err := h.aiConfigRepo.SetActiveConfig(ctx, req.ProviderName)
	if err != nil {
		if errors.Is(err, repository.ErrAIConfigNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": "Provider name not found: " + req.ProviderName,
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update active AI provider: " + err.Error(),
			},
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"message":       "Active AI provider updated successfully",
			"provider_name": req.ProviderName,
		},
		"error": nil,
	})
}
