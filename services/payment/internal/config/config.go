package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL    string
	GRPCPort string
	NatsURL  string
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func getOrDefault(key, defaultVal string) string {
	val := getEnv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func mustGetEnv(key string) (string, error) {
	val := getEnv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable: %s is not set", key)
	}
	return val, nil
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbURL, err := mustGetEnv("DB_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	grpcPort := getOrDefault("GRPC_PORT", "50051")
	natsURL := getOrDefault("NATS_URL", "nats://localhost:4222")
	return &Config{
		DBURL:    dbURL,
		GRPCPort: grpcPort,
		NatsURL:  natsURL,
	}, nil
}
