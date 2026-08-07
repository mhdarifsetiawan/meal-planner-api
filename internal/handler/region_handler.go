package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type City struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ProvinceName string `json:"province_name"`
}

type RegionHandler struct {
	db *pgxpool.Pool
}

func NewRegionHandler(db *pgxpool.Pool) *RegionHandler {
	return &RegionHandler{db: db}
}

func (h *RegionHandler) HandleGetCities(c *fiber.Ctx) error {
	rows, err := h.db.Query(context.Background(), `
		SELECT c.id, c.name, p.name AS province_name
		FROM cities c
		JOIN provinces p ON c.province_id = p.id
		ORDER BY p.name, c.name
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"data":  nil,
			"error": fiber.Map{"message": "Failed to fetch cities: " + err.Error()},
		})
	}
	defer rows.Close()

	var cities []City
	for rows.Next() {
		var city City
		if err := rows.Scan(&city.ID, &city.Name, &city.ProvinceName); err != nil {
			continue
		}
		cities = append(cities, city)
	}

	if cities == nil {
		cities = []City{}
	}

	return c.JSON(fiber.Map{
		"data":  fiber.Map{"cities": cities},
		"error": nil,
	})
}
