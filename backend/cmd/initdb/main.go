package main

import (
	"fmt"
	"log"

	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"
)

func main() {
	cfg := config.LoadConfig()
	database, err := db.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("Unable to access database connection: %v\n", err)
	}
	defer sqlDB.Close()

	if err := db.EnsureSchema(database); err != nil {
		log.Fatalf("Unable to ensure schema: %v\n", err)
	}

	fmt.Println("Schema initialized successfully!")
}

