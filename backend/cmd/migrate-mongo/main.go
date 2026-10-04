package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/divyanshmehta355/aurahub/backend/internal/mongomigrate"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	apply := flag.Bool("apply", false, "write imported records to PostgreSQL (default is read-only dry run)")
	flag.Parse()

	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Unable to load .env: %v", err)
	}
	mongoURI := os.Getenv("MONGO_URI")
	databaseURL := os.Getenv("DATABASE_URL")
	if mongoURI == "" || databaseURL == "" {
		log.Fatal("MONGO_URI and DATABASE_URL must both be set")
	}
	databaseName, err := mongoDatabaseName(mongoURI, os.Getenv("MONGO_DATABASE"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Unable to connect to MongoDB source; verify MONGO_URI, network access, and credentials")
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		if err := client.Disconnect(disconnectCtx); err != nil {
			log.Printf("Unable to close MongoDB source connection: %v", err)
		}
	}()
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Unable to reach MongoDB source; verify network access and credentials")
	}

	var target *gorm.DB
	if *apply {
		target, err = db.OpenPostgres(databaseURL)
		if err != nil {
			log.Fatal("Unable to connect to PostgreSQL target; verify DATABASE_URL and database availability")
		}
		sqlDB, err := target.DB()
		if err != nil {
			log.Fatal("Unable to access PostgreSQL connection")
		}
		defer sqlDB.Close()
	}

	report, err := mongomigrate.Run(context.Background(), client.Database(databaseName), target, *apply)
	if err != nil {
		log.Fatalf("MongoDB import failed: %v", err)
	}
	if *apply {
		if err := printPostgresCounts(target); err != nil {
			log.Fatalf("Import committed, but PostgreSQL row counts could not be verified: %v", err)
		}
		fmt.Println("Import completed. MongoDB was read only; PostgreSQL writes were committed atomically.")
	} else {
		fmt.Println("Dry run completed. MongoDB was read only; PostgreSQL was not modified.")
	}
	collections := make([]string, 0, len(report))
	for collection := range report {
		collections = append(collections, collection)
	}
	sort.Strings(collections)
	for _, collection := range collections {
		fmt.Printf("%-32s %d processed\n", collection, report[collection])
	}
}

func printPostgresCounts(target *gorm.DB) error {
	tables := []string{
		"users",
		"videos",
		"comments",
		"notifications",
		"playlists",
		"playlist_videos",
		"subscriptions",
		"user_activities",
		"watch_history",
		"watch_later",
	}
	silentTarget := target.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	for _, table := range tables {
		var count int64
		if err := silentTarget.Table(table).Count(&count).Error; err != nil {
			return fmt.Errorf("count PostgreSQL table %s: %w", table, err)
		}
		fmt.Printf("PostgreSQL %-20s %d rows total\n", table, count)
	}
	return nil
}

func mongoDatabaseName(uri, fallback string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("MONGO_URI is invalid")
	}
	name := strings.Trim(parsed.Path, "/")
	if name == "" {
		name = strings.TrimSpace(fallback)
	}
	if name == "" {
		return "", fmt.Errorf("MongoDB database is missing from MONGO_URI; set MONGO_DATABASE")
	}
	if strings.Contains(name, "/") {
		return "", fmt.Errorf("MongoDB database name is invalid")
	}
	return name, nil
}
