package handler

import (
	"net/http"
	"strconv"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/job"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type AdminPriceWatchHandler struct {
	repo         repository.PriceWatchRepository
	consensusJob *job.ConsensusJob
}

func NewAdminPriceWatchHandler(repo repository.PriceWatchRepository, consensusJob *job.ConsensusJob) *AdminPriceWatchHandler {
	return &AdminPriceWatchHandler{
		repo:         repo,
		consensusJob: consensusJob,
	}
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

type runConsensusRequest struct {
	MinSubmissions   int     `json:"min_submissions"`
	TolerancePercent float64 `json:"tolerance_percent"`
}

func (h *AdminPriceWatchHandler) HandleRunConsensus(c *fiber.Ctx) error {
	if h.consensusJob == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Consensus job is not initialized",
			},
		})
	}

	var req runConsensusRequest
	if err := c.BodyParser(&req); err != nil {
		// Body bisa kosong, gunakan default
		req = runConsensusRequest{}
	}

	minSubs := req.MinSubmissions
	if minSubs <= 0 {
		minSubs = 3 // default
	}
	tolerance := req.TolerancePercent
	if tolerance <= 0 {
		tolerance = 15.0 // default
	}

	summary, err := h.consensusJob.RunConsensusValidation(c.Context(), minSubs, tolerance)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Consensus job execution failed: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  summary,
		"error": nil,
	})
}

func (h *AdminPriceWatchHandler) HandleGetAllSubmissions(c *fiber.Ctx) error {
	var cityIDPtr *int
	if cityStr := c.Query("city_id"); cityStr != "" && cityStr != "all" {
		if id, err := strconv.Atoi(cityStr); err == nil && id > 0 {
			cityIDPtr = &id
		}
	}

	var statusPtr *string
	if statusStr := c.Query("status"); statusStr != "" && statusStr != "all" {
		statusPtr = &statusStr
	}

	subs, err := h.repo.GetAllSubmissions(c.Context(), cityIDPtr, statusPtr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch submissions: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  subs,
		"error": nil,
	})
}
