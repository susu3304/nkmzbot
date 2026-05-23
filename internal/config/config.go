package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

	// IMM
	IMMBinaryPath     string
	IMMTimeout        time.Duration
	IMMMaxSourceBytes int
	IMMMaxOutputBytes int
}

func Load() (*Config, error) {
	// Load environment variables from .env if present (non-fatal if missing)
	_ = godotenv.Load()

	cfg := &Config{
		DiscordToken:      os.Getenv("DISCORD_TOKEN"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		APIURL:            getEnvDefault("API_URL", "http://localhost:3000"),
		APIToken:          os.Getenv("API_TOKEN"),
		IMMBinaryPath:     getEnvDefault("IMM_BINARY", "imm"),
		IMMTimeout:        getDurationEnvDefault("IMM_TIMEOUT_MS", 3*time.Second),
		IMMMaxSourceBytes: getIntEnvDefault("IMM_MAX_SOURCE_BYTES", 64*1024),
		IMMMaxOutputBytes: getIntEnvDefault("IMM_MAX_OUTPUT_BYTES", 64*1024),
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

func getIntEnvDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func getDurationEnvDefault(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return time.Duration(parsed) * time.Millisecond
}
