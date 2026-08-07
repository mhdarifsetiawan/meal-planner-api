package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env (opsional di production — env vars sudah diset langsung)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	app := fiber.New(fiber.Config{
		AppName: "MasakApa API v0.1.0",
		// Semua error response menggunakan format {"data":null,"error":{"message":"..."}}
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"data": nil,
				"error": fiber.Map{
					"message": err.Error(),
				},
			})
		},
	})

	// Middleware global
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
	}))

	// ---------- Routes ----------

	// Health check — tidak butuh auth
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"status":  "ok",
				"service": "masakapa-api",
			},
			"error": nil,
		})
	})

	// TODO: daftarkan route grup di sini seiring development
	// api := app.Group("/api/v1")
	// handler.RegisterAuthRoutes(api, ...)
	// handler.RegisterMenuRoutes(api, ...)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
