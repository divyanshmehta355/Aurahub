package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	ValkeyURL      string
	Port           string
	JWTSecret      string
	AllowedOrigins string
}

func LoadConfig() *Config {
	err := godotenv.Load(".env") // Fallback for local dev
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/aurahub?sslmode=disable"
	}

	valkeyUrl := os.Getenv("VALKEY_URL")
	if valkeyUrl == "" {
		valkeyUrl = "redis://localhost:6379/0"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	allowedOrigins := os.Getenv("FRONTEND_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}

	return &Config{
		DatabaseURL:    dbUrl,
		ValkeyURL:      valkeyUrl,
		Port:           port,
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AllowedOrigins: allowedOrigins,
	}
}
