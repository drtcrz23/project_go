package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	GRPCPort string
	LogLevel string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	config := &Config{
		GRPCPort: ":50051",
		LogLevel: "info",
	}

	if port := os.Getenv("GRPC_PORT"); port != "" {
		config.GRPCPort = port
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}

	return config, nil
}
