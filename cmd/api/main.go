package main

import (
	"context"
	"log"
	"os"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/handler"
	"meal-planner-api/internal/payment"
	"meal-planner-api/internal/price"
	"meal-planner-api/internal/repository"
	"meal-planner-api/internal/subscription"

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

	// Repositories & Services
	userRepo := repository.NewUserRepository(dbPool)
	menuRepo := repository.NewMenuRepository(dbPool)
	shoppingListRepo := repository.NewShoppingListRepository(dbPool)
	historyRepo := repository.NewHistoryRepository(dbPool)
	subRepo := repository.NewSubscriptionRepository(dbPool)
	rateLimiter := subscription.NewMemoryRateLimiter()
	priceProvider := price.NewAIEstimateProvider(dbPool)
	dummyPayment := payment.NewDummyPaymentProvider(0)

	// AI Provider Initialization (Default: OpenAI)
	var aiProvider ai.AIProvider
	openAIKey := os.Getenv("AI_PROVIDER_API_KEY_OPENAI")
	if openAIKey != "" {
		provider, err := ai.NewOpenAIProvider(openAIKey, "")
		if err != nil {
			log.Printf("Warning: OpenAI provider init error: %v", err)
		} else {
			aiProvider = provider
			log.Println("🤖 OpenAI Provider initialized")
		}
	}

	// Handlers
	onboardingHandler := handler.NewOnboardingHandler(userRepo)
	menuSelectHandler := handler.NewMenuSelectHandler(menuRepo)
	shoppingListHandler := handler.NewShoppingListHandler(shoppingListRepo)
	historyHandler := handler.NewHistoryHandler(historyRepo)
	subHandler := handler.NewSubscriptionHandler(subRepo, dummyPayment)
	var menuHandler *handler.MenuHandler
	if aiProvider != nil {
		menuHandler = handler.NewMenuHandler(aiProvider, priceProvider, userRepo, subRepo, rateLimiter)
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

	api := app.Group("/api/v1")

	// Profile route
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

	// Menu select endpoint
	api.Post("/menu/select", auth.RequireAuth(), menuSelectHandler.HandleSelectMenu)

	// Shopping list endpoints
	api.Get("/shopping-list/:id", auth.RequireAuth(), shoppingListHandler.HandleGetShoppingList)
	api.Patch("/shopping-list/:id/item", auth.RequireAuth(), shoppingListHandler.HandleUpdateShoppingListItem)

	// History endpoint
	api.Get("/history", auth.RequireAuth(), historyHandler.HandleGetHistory)

	// Subscription endpoint
	api.Post("/subscription/subscribe", auth.RequireAuth(), subHandler.HandleSubscribe)

	// Menu generate endpoint
	if menuHandler != nil {
		api.Post("/menu/generate", auth.RequireAuth(), menuHandler.HandleGenerateMenu)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
