package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Discord Bot
	DiscordToken string

	// Database
	DatabaseURL string

	// API
	APIURL   string
	APIToken string
}

func Load() (*Config, error) {
	// Load environment variables from .env if present (non-fatal if missing)
	_ = godotenv.Load()

	cfg := &Config{
		DiscordToken:        os.Getenv("DISCORD_TOKEN"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		APIURL:              getEnvDefault("API_URL", "http://localhost:3000"),
		APIToken:            os.Getenv("API_TOKEN"),
	}

	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN is required")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
