package main

import (
	"context"
	"log"
	"os"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/handler"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env (opsional di production — env vars sudah diset langsung)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Database Connection Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5434/masakapa?sslmode=disable"
	}

	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Printf("Warning: Failed to create DB pool: %v", err)
	} else {
		defer dbPool.Close()
		if err := dbPool.Ping(ctx); err != nil {
			log.Printf("Warning: DB ping failed: %v", err)
		} else {
			log.Println("✅ Connected to PostgreSQL database")
		}
	}

	// Repositories
	userRepo := repository.NewUserRepository(dbPool)

	// Handlers
	onboardingHandler := handler.NewOnboardingHandler(userRepo)

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

	api := app.Group("/api/v1")

	// Pre-route me / profile untuk test auth middleware
	api.Get("/me", auth.RequireAuth(), func(c *fiber.Ctx) error {
		userID, _ := auth.GetUserID(c)
		email := auth.GetUserEmail(c)
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"user_id": userID,
				"email":   email,
			},
			"error": nil,
		})
	})

	// Onboarding endpoint
	api.Post("/onboarding", auth.RequireAuth(), onboardingHandler.HandleOnboarding)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
