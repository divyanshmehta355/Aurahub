package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	ValkeyURL           string
	Port                string
	JWTSecret           string
	AllowedOrigins      string
	KafkaBrokers        []string
	KafkaUser           string
	KafkaPassword       string
	KafkaAuthMethod     string
	KafkaCACertPath     string
	KafkaClientCertPath string
	KafkaClientKeyPath  string
	KafkaCACert         string
	KafkaClientCert     string
	KafkaClientKey      string
	KafkaInsecureSkip   bool
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

	var kafkaBrokers []string
	if rawBrokers := os.Getenv("KAFKA_BROKERS"); rawBrokers != "" {
		for _, b := range strings.Split(rawBrokers, ",") {
			trimmed := strings.TrimSpace(b)
			if trimmed != "" {
				kafkaBrokers = append(kafkaBrokers, trimmed)
			}
		}
	}

	authMethod := os.Getenv("KAFKA_AUTH_METHOD")
	if authMethod == "" {
		authMethod = "scram-sha-256"
	}

	insecureSkip := os.Getenv("KAFKA_INSECURE_SKIP_VERIFY") == "true" || os.Getenv("KAFKA_INSECURE_SKIP_VERIFY") == "1"
	// If credentials are provided but no CA certs are configured, default to true for zero-config cloud SASL
	if (os.Getenv("KAFKA_USER") != "" && os.Getenv("KAFKA_PASSWORD") != "") &&
		os.Getenv("KAFKA_CA_CERT_PATH") == "" && os.Getenv("KAFKA_CA_CERT") == "" {
		insecureSkip = true
	}

	return &Config{
		DatabaseURL:         dbUrl,
		ValkeyURL:           valkeyUrl,
		Port:                port,
		JWTSecret:           os.Getenv("JWT_SECRET"),
		AllowedOrigins:      allowedOrigins,
		KafkaBrokers:        kafkaBrokers,
		KafkaUser:           os.Getenv("KAFKA_USER"),
		KafkaPassword:       os.Getenv("KAFKA_PASSWORD"),
		KafkaAuthMethod:     authMethod,
		KafkaCACertPath:     os.Getenv("KAFKA_CA_CERT_PATH"),
		KafkaClientCertPath: os.Getenv("KAFKA_CLIENT_CERT_PATH"),
		KafkaClientKeyPath:  os.Getenv("KAFKA_CLIENT_KEY_PATH"),
		KafkaCACert:         os.Getenv("KAFKA_CA_CERT"),
		KafkaClientCert:     os.Getenv("KAFKA_CLIENT_CERT"),
		KafkaClientKey:      os.Getenv("KAFKA_CLIENT_KEY"),
		KafkaInsecureSkip:   insecureSkip,
	}
}
