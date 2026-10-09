package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/api"
	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"

	"github.com/divyanshmehta355/aurahub/backend/internal/kafka"
	"github.com/divyanshmehta355/aurahub/backend/internal/kafka/workers"
	"github.com/divyanshmehta355/aurahub/backend/internal/opensearch"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

	// Configure connection pool boundaries for production stability
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

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

	app.Use(recover.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	}))

	var serverName string
	if len(cfg.KafkaBrokers) > 0 {
		host, _, err := net.SplitHostPort(cfg.KafkaBrokers[0])
		if err == nil {
			serverName = host
		} else {
			serverName = cfg.KafkaBrokers[0]
		}
	}

	// Initialize Kafka Producer and background event workers with TLS / SASL authentication
	kafkaAuth := kafka.AuthConfig{
		User:               cfg.KafkaUser,
		Password:           cfg.KafkaPassword,
		AuthMethod:         cfg.KafkaAuthMethod,
		ServerName:         serverName,
		CACertPath:         cfg.KafkaCACertPath,
		ClientCertPath:     cfg.KafkaClientCertPath,
		ClientKeyPath:      cfg.KafkaClientKeyPath,
		CACert:             cfg.KafkaCACert,
		ClientCert:         cfg.KafkaClientCert,
		ClientKey:          cfg.KafkaClientKey,
		InsecureSkipVerify: cfg.KafkaInsecureSkip,
	}
	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, kafkaAuth)
	defer kafkaProducer.Close()

	// Initialize OpenSearch client (Phase 3)
	var osAddresses []string
	if cfg.OpenSearchURL != "" {
		osAddresses = []string{cfg.OpenSearchURL}
	}
	osClient, err := opensearch.NewClient(opensearch.Config{
		Addresses:          osAddresses,
		Username:           cfg.OpenSearchUser,
		Password:           cfg.OpenSearchPassword,
		InsecureSkipVerify: cfg.OpenSearchInsecure,
	})
	if err != nil {
		log.Printf("[OpenSearch] Warning: Initialization error: %v\n", err)
	}

	var workerWg sync.WaitGroup
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	if osClient != nil && osClient.Enabled() {
		go func() {
			if err := osClient.EnsureIndex(workerCtx); err != nil {
				log.Printf("[OpenSearch] EnsureIndex warning: %v\n", err)
			}
		}()
	}

	if len(cfg.KafkaBrokers) > 0 {
		go kafka.EnsureTopics(workerCtx, cfg.KafkaBrokers, kafkaAuth,
			kafka.TopicVideoViews,
			kafka.TopicBatchUploadJobs,
			kafka.TopicNotifications,
			kafka.TopicVideoLifecycle,
		)
		go workers.StartViewsFlusher(workerCtx, cfg.KafkaBrokers, kafkaAuth, repository, &workerWg)
		go workers.StartBatchUploadWorker(workerCtx, cfg.KafkaBrokers, kafkaAuth, repository, rdb, kafkaProducer, &workerWg)
		if osClient != nil && osClient.Enabled() {
			go workers.StartOpenSearchIndexer(workerCtx, cfg.KafkaBrokers, kafkaAuth, repository, osClient, &workerWg)
		}
	}

	// Pass the PostgreSQL query layer, Valkey client, Kafka producer, and OpenSearch client to handlers.
	server := api.NewServer(repository, rdb, kafkaProducer, osClient)
	server.SetupRoutes(app)

	// Graceful shutdown listener on SIGINT / SIGTERM
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		log.Println("Received shutdown signal. Gracefully stopping Aurahub server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Error shutting down HTTP server: %v\n", err)
		}
	}()

	log.Printf("Starting server on port %s...", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Printf("HTTP listener closed: %v\n", err)
	}

	log.Println("Draining background Kafka workers...")
	cancelWorkers()
	workerWg.Wait()
	log.Println("Aurahub server and workers shutdown complete.")
}
