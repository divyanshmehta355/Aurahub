package main

import (
	"context"
	"log"

	"github.com/divyanshmehta355/aurahub/backend/api"
	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.LoadConfig()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set before starting the API")
	}

	// Initialize the PostgreSQL connection through GORM.
	database, err := db.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("Unable to access database connection: %v", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(context.Background()); err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	repository := db.NewRepository(database)

	// Initialize Valkey using its Redis-compatible protocol.
	opt, err := redis.ParseURL(cfg.ValkeyURL)
	if err != nil {
		log.Fatalf("Invalid Valkey URL: %v\n", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	// Ensure Valkey connection is alive
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Could not connect to Valkey: %v", err)
	}

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		AppName:   "Aurahub API",
		BodyLimit: 8 * 1024 * 1024,
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	// Pass the PostgreSQL query layer and Valkey client to handlers.
	server := api.NewServer(repository, rdb)
	server.SetupRoutes(app)

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
