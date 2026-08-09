package handler

import (
	"net/http"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type PriceWatchSubmissionHandler struct {
	pwRepo   repository.PriceWatchRepository
	userRepo repository.UserRepository
}

func NewPriceWatchSubmissionHandler(pwRepo repository.PriceWatchRepository, userRepo repository.UserRepository) *PriceWatchSubmissionHandler {
	return &PriceWatchSubmissionHandler{
		pwRepo:   pwRepo,
		userRepo: userRepo,
	}
}

type CreateSubmissionRequest struct {
	WatchItemID    int  `json:"watch_item_id"`
	SubmittedPrice int  `json:"submitted_price"`
	CityID         *int `json:"city_id"`
}

func (h *PriceWatchSubmissionHandler) HandleSubmitPrice(c *fiber.Ctx) error {
	userID, err := auth.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Unauthorized: missing user_id in context",
			},
		})
	}

	var req CreateSubmissionRequest
	if err := c.BodyParser(&req); err != nil || req.WatchItemID <= 0 || req.SubmittedPrice <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: watch_item_id and submitted_price (> 0) are required",
			},
		})
	}

	ctx := c.Context()

	// 1. Resolve City ID
	var cityID int
	if req.CityID != nil && *req.CityID > 0 {
		cityID = *req.CityID
	} else if h.userRepo != nil {
		user, uErr := h.userRepo.GetUserByID(ctx, userID)
		if uErr == nil && user != nil && user.CityID != nil && *user.CityID > 0 {
			cityID = *user.CityID
		}
	}

	if cityID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "city_id is required: please set city_id in request or update your profile",
			},
		})
	}

	// 2. Verify Watch Item exists
	item, err := h.pwRepo.GetItemByID(ctx, req.WatchItemID)
	if err != nil || item == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Price watch item not found",
			},
		})
	}

	// 3. Check for 24-hour anti-duplication rule
	hasRecent, err := h.pwRepo.HasRecentSubmission(ctx, userID, req.WatchItemID, cityID)
	if err == nil && hasRecent {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Anda sudah memberikan laporan harga untuk bahan ini di kota ini dalam 24 jam terakhir.",
			},
		})
	}

	// 4. Create Submission
	sub := &model.PriceSubmission{
		WatchItemID:    req.WatchItemID,
		UserID:         userID,
		CityID:         cityID,
		SubmittedPrice: req.SubmittedPrice,
	}

	if err := h.pwRepo.CreateSubmission(ctx, sub); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to record price submission: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  sub,
		"error": nil,
	})
}
