package handler

import (
	"net/http"
	"strconv"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type AdminPriceWatchHandler struct {
	repo repository.PriceWatchRepository
}

func NewAdminPriceWatchHandler(repo repository.PriceWatchRepository) *AdminPriceWatchHandler {
	return &AdminPriceWatchHandler{repo: repo}
}

type CreateCampaignRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type CreateItemRequest struct {
	IngredientName string  `json:"ingredient_name"`
	Unit           string  `json:"unit"`
	IconURL        *string `json:"icon_url"`
	DisplayOrder   int     `json:"display_order"`
	IsActive       *bool   `json:"is_active"`
}

func (h *AdminPriceWatchHandler) HandleCreateCampaign(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	var req CreateCampaignRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: title is required",
			},
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	camp := &model.PriceWatchCampaign{
		Title:       req.Title,
		Description: req.Description,
		IsActive:    isActive,
		CreatedBy:   userID,
	}

	if err := h.repo.CreateCampaign(c.Context(), camp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to create campaign: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  camp,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleGetCampaigns(c *fiber.Ctx) error {
	includeInactive := c.Query("include_inactive", "true") == "true"
	campaigns, err := h.repo.GetCampaigns(c.Context(), includeInactive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch campaigns: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  campaigns,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleGetCampaignByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid campaign ID",
			},
		})
	}

	camp, err := h.repo.GetCampaignByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Campaign not found: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  camp,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleUpdateCampaign(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid campaign ID",
			},
		})
	}

	var req CreateCampaignRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: title is required",
			},
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	camp := &model.PriceWatchCampaign{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		IsActive:    isActive,
	}

	if err := h.repo.UpdateCampaign(c.Context(), camp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update campaign: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  camp,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleDeleteCampaign(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid campaign ID",
			},
		})
	}

	if err := h.repo.DeleteCampaign(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to delete campaign: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  fiber.Map{"message": "Campaign deleted successfully"},
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleCreateItem(c *fiber.Ctx) error {
	campIDStr := c.Params("id")
	campID, err := strconv.Atoi(campIDStr)
	if err != nil || campID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid campaign ID",
			},
		})
	}

	var req CreateItemRequest
	if err := c.BodyParser(&req); err != nil || req.IngredientName == "" || req.Unit == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: ingredient_name and unit are required",
			},
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	item := &model.PriceWatchItem{
		CampaignID:     campID,
		IngredientName: req.IngredientName,
		Unit:           req.Unit,
		IconURL:        req.IconURL,
		DisplayOrder:   req.DisplayOrder,
		IsActive:       isActive,
	}

	if err := h.repo.CreateItem(c.Context(), item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to create item: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  item,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleUpdateItem(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid item ID",
			},
		})
	}

	var req CreateItemRequest
	if err := c.BodyParser(&req); err != nil || req.IngredientName == "" || req.Unit == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: ingredient_name and unit are required",
			},
		})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	item := &model.PriceWatchItem{
		ID:             id,
		IngredientName: req.IngredientName,
		Unit:           req.Unit,
		IconURL:        req.IconURL,
		DisplayOrder:   req.DisplayOrder,
		IsActive:       isActive,
	}

	if err := h.repo.UpdateItem(c.Context(), item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update item: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  item,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleDeleteItem(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid item ID",
			},
		})
	}

	if err := h.repo.DeleteItem(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to delete item: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  fiber.Map{"message": "Item deleted successfully"},
		"error": nil,
	})
}
