package main

import (
	"context"
	"fmt"
	"log"

	"github.com/divyanshmehta355/aurahub/backend/config"
	"github.com/divyanshmehta355/aurahub/backend/db"
	"github.com/jackc/pgx/v5/pgtype"
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

	repository := db.NewRepository(database)
	user, err := repository.CreateUser(context.Background(), db.CreateUserParams{
		Email:    "debug123@test.com",
		Password: "password123",
		Username: "debug123",
		Avatar:   pgtype.Text{String: "/api/avatar/debug123", Valid: true},
	})

	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: %+v\n", user)
	}
}
