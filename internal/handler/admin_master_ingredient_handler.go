package handler

import (
	"net/http"
	"strconv"
	"strings"

	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type AdminMasterIngredientHandler struct {
	repo repository.MasterIngredientRepository
}

func NewAdminMasterIngredientHandler(repo repository.MasterIngredientRepository) *AdminMasterIngredientHandler {
	return &AdminMasterIngredientHandler{repo: repo}
}

func (h *AdminMasterIngredientHandler) HandleGetAll(c *fiber.Ctx) error {
	category := c.Query("category")
	search := c.Query("search")

	items, err := h.repo.GetAll(c.Context(), category, search)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to fetch master ingredients: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  items,
		"error": nil,
	})
}

func (h *AdminMasterIngredientHandler) HandleCreate(c *fiber.Ctx) error {
	var req model.CreateMasterIngredientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Category) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Name and category are required",
			},
		})
	}

	if req.DefaultUnit == "" {
		req.DefaultUnit = "kg"
	}

	item, err := h.repo.Create(c.Context(), req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to create master ingredient: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  item,
		"error": nil,
	})
}

func (h *AdminMasterIngredientHandler) HandleUpdate(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid master ingredient ID",
			},
		})
	}

	var req model.UpdateMasterIngredientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid request body: " + err.Error(),
			},
		})
	}

	item, err := h.repo.Update(c.Context(), id, req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to update master ingredient: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  item,
		"error": nil,
	})
}

func (h *AdminMasterIngredientHandler) HandleDelete(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid master ingredient ID",
			},
		})
	}

	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to delete master ingredient: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  fiber.Map{"message": "Master ingredient deleted successfully"},
		"error": nil,
	})
}

func (h *AdminMasterIngredientHandler) HandleAddAlias(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid master ingredient ID",
			},
		})
	}

	var req model.AddAliasRequest
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.AliasName) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Alias name is required",
			},
		})
	}

	alias, err := h.repo.AddAlias(c.Context(), id, req.AliasName)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to add alias (might already exist): " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"data":  alias,
		"error": nil,
	})
}

func (h *AdminMasterIngredientHandler) HandleDeleteAlias(c *fiber.Ctx) error {
	aliasIDParam := c.Params("alias_id")
	aliasID, err := strconv.Atoi(aliasIDParam)
	if err != nil || aliasID <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Invalid alias ID",
			},
		})
	}

	if err := h.repo.DeleteAlias(c.Context(), aliasID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"data": nil,
			"error": fiber.Map{
				"message": "Failed to delete alias: " + err.Error(),
			},
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  fiber.Map{"message": "Alias deleted successfully"},
		"error": nil,
	})
}
