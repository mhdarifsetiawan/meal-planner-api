package main

import (
	"context"
	"log"
	"os"

	"meal-planner-api/internal/ai"
	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/handler"
	"meal-planner-api/internal/job"
	"meal-planner-api/internal/payment"
	"meal-planner-api/internal/price"
	"meal-planner-api/internal/repository"
	"meal-planner-api/internal/subscription"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5"
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
	var dbPool *pgxpool.Pool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Printf("Warning: Failed to parse DB URL: %v", err)
	} else {
		// Disable prepared statement caching to support PgBouncer transaction mode / Supabase Connection Pooler
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

		dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			log.Printf("Warning: Failed to create DB pool: %v", err)
		} else {
			defer dbPool.Close()
			if err := dbPool.Ping(ctx); err != nil {
				log.Printf("Warning: DB ping failed: %v", err)
			} else {
				log.Println("✅ Connected to PostgreSQL database (PgBouncer compatible)")
			}
		}
	}

	// Repositories & Services
	userRepo := repository.NewUserRepository(dbPool)
	menuRepo := repository.NewMenuRepository(dbPool)
	shoppingListRepo := repository.NewShoppingListRepository(dbPool)
	historyRepo := repository.NewHistoryRepository(dbPool)
	subRepo := repository.NewSubscriptionRepository(dbPool)
	aiConfigRepo := repository.NewAIConfigRepository(dbPool)
	rateLimiter := subscription.NewMemoryRateLimiter()
	priceProvider := price.NewAIEstimateProvider(dbPool)
	dummyPayment := payment.NewDummyPaymentProvider(0)

	// Dynamic AI Provider (reads active provider from DB)
	dynamicAIProvider := ai.NewDynamicAIProvider(aiConfigRepo)

	// Handlers
	onboardingHandler := handler.NewOnboardingHandler(userRepo)
	menuSelectHandler := handler.NewMenuSelectHandler(menuRepo)
	shoppingListHandler := handler.NewShoppingListHandler(shoppingListRepo)
	historyHandler := handler.NewHistoryHandler(historyRepo)
	subHandler := handler.NewSubscriptionHandler(subRepo, dummyPayment)
	regionHandler := handler.NewRegionHandler(dbPool)
	adminAIConfigHandler := handler.NewAdminAIConfigHandler(aiConfigRepo)

	menuHandler := handler.NewMenuHandler(dynamicAIProvider, priceProvider, userRepo, subRepo, rateLimiter)

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
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:3000,http://localhost:3001,https://masakapa-api.fly.dev,https://*.vercel.app",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))
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
	api.Get("/me", auth.RequireAuth(userRepo), func(c *fiber.Ctx) error {
		userID, _ := auth.GetUserID(c)
		email := auth.GetUserEmail(c)
		role, _ := c.Locals(auth.LocalUserRoleKey).(string)
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"user_id": userID,
				"email":   email,
				"role":    role,
			},
			"error": nil,
		})
	})

	// Public region endpoints (no auth needed for city list)
	api.Get("/cities", regionHandler.HandleGetCities)

	// Onboarding & Preferences endpoints
	api.Post("/onboarding", auth.RequireAuth(userRepo), onboardingHandler.HandleOnboarding)
	api.Get("/preferences", auth.RequireAuth(userRepo), onboardingHandler.HandleGetPreferences)

	// Menu select endpoint
	api.Post("/menu/select", auth.RequireAuth(userRepo), menuSelectHandler.HandleSelectMenu)

	// Shopping list endpoints
	api.Get("/shopping-list/:id", auth.RequireAuth(userRepo), shoppingListHandler.HandleGetShoppingList)
	api.Patch("/shopping-list/:id/item", auth.RequireAuth(userRepo), shoppingListHandler.HandleUpdateShoppingListItem)

	// History endpoints
	api.Get("/history", auth.RequireAuth(userRepo), historyHandler.HandleGetHistory)
	api.Delete("/history/:id", auth.RequireAuth(userRepo), historyHandler.HandleDeleteHistory)

	// Subscription endpoint
	api.Post("/subscription/subscribe", auth.RequireAuth(userRepo), subHandler.HandleSubscribe)

	// Price Watch Admin endpoints
	pwRepo := repository.NewPriceWatchRepository(dbPool)
	consensusJob := job.NewConsensusJob(dbPool)
	adminPWHandler := handler.NewAdminPriceWatchHandler(pwRepo, consensusJob)

	admin := api.Group("/admin", auth.RequireAuth(userRepo))
	admin.Get("/ai-config", adminAIConfigHandler.HandleGetConfigs)
	admin.Post("/ai-config", adminAIConfigHandler.HandleSelectConfig)

	admin.Get("/price-watch/campaigns", adminPWHandler.HandleGetCampaigns)
	admin.Post("/price-watch/campaigns", adminPWHandler.HandleCreateCampaign)
	admin.Put("/price-watch/campaigns/:id", adminPWHandler.HandleUpdateCampaign)
	admin.Delete("/price-watch/campaigns/:id", adminPWHandler.HandleDeleteCampaign)
	admin.Post("/price-watch/items", adminPWHandler.HandleCreateItem)
	admin.Put("/price-watch/items/:item_id", adminPWHandler.HandleUpdateItem)
	admin.Delete("/price-watch/items/:item_id", adminPWHandler.HandleDeleteItem)
	admin.Post("/price-watch/run-consensus", adminPWHandler.HandleRunConsensus)
	admin.Get("/price-watch/submissions", adminPWHandler.HandleGetAllSubmissions)

	// Master Ingredients Admin endpoints
	masterIngredientRepo := repository.NewMasterIngredientRepository(dbPool)
	adminMasterIngredientHandler := handler.NewAdminMasterIngredientHandler(masterIngredientRepo)

	admin.Get("/master-ingredients", adminMasterIngredientHandler.HandleGetAll)
	admin.Post("/master-ingredients", adminMasterIngredientHandler.HandleCreate)
	admin.Put("/master-ingredients/:id", adminMasterIngredientHandler.HandleUpdate)
	admin.Delete("/master-ingredients/:id", adminMasterIngredientHandler.HandleDelete)
	admin.Post("/master-ingredients/:id/aliases", adminMasterIngredientHandler.HandleAddAlias)
	admin.Delete("/master-ingredients/aliases/:alias_id", adminMasterIngredientHandler.HandleDeleteAlias)

	// Price Watch User endpoints
	userPWHandler := handler.NewUserPriceWatchHandler(pwRepo)
	pwSubHandler := handler.NewPriceWatchSubmissionHandler(pwRepo, userRepo)

	api.Get("/price-watch/campaigns/active", auth.RequireAuth(userRepo), userPWHandler.HandleGetActiveCampaigns)
	api.Get("/price-watch/submissions/me", auth.RequireAuth(userRepo), userPWHandler.HandleGetUserSubmissions)
	api.Post("/price-watch/submissions", auth.RequireAuth(userRepo), pwSubHandler.HandleSubmitPrice)

	// User Credits endpoint
	creditRepo := repository.NewCreditRepository(dbPool)
	creditHandler := handler.NewCreditHandler(creditRepo)
	api.Get("/credits/me", auth.RequireAuth(userRepo), creditHandler.HandleGetMyCredits)

	// Menu generate endpoint
	if menuHandler != nil {
		api.Post("/menu/generate", auth.RequireAuth(userRepo), menuHandler.HandleGenerateMenu)
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
