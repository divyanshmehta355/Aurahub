package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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

	schema, err := os.ReadFile("db/schema.sql")
	if err != nil {
		log.Fatalf("Unable to read schema: %v\n", err)
	}

	enumStatements := []string{
		`DO $$ BEGIN CREATE TYPE video_visibility AS ENUM ('public', 'unlisted', 'private'); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`DO $$ BEGIN CREATE TYPE streamtape_status AS ENUM ('active', 'dead', 'pending'); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`DO $$ BEGIN CREATE TYPE notification_type AS ENUM ('like', 'comment', 'reply', 'new_video'); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`DO $$ BEGIN CREATE TYPE interaction_type AS ENUM ('view', 'like'); EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}
	for _, statement := range enumStatements {
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Unable to ensure database enum exists: %v\n", err)
		}
	}

	for _, statement := range strings.Split(string(schema), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Unable to apply schema statement: %v\n", err)
		}
	}

	fmt.Println("Schema initialized successfully!")
}
