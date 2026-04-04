package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBUrl  string
	Port   string
	Env    string
	APIKey string
}

// Load reads a .env file (if present) then populates Config from env vars.
// Returns an error if required fields are missing.
func Load() (*Config, error) {
	// Best-effort load; ignore error if .env is absent (vars may come from OS env).
	_ = godotenv.Load(".env")

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		return nil, errors.New("DATABASE_URL is required but not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, errors.New("API_KEY is required but not set")
	}

	return &Config{
		DBUrl:  dbUrl,
		Port:   port,
		Env:    env,
		APIKey: apiKey,
	}, nil
}
