package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl  string
	Port   string
	Env    string
	APIKey string
}

func Load() (Config, error) {
	_ = godotenv.Load(".env") // ignore error: .env may not exist when env vars are injected externally

	cfg := Config{
		DBUrl:  os.Getenv("DATABASE_URL"),
		Port:   os.Getenv("PORT"),
		Env:    os.Getenv("ENV"),
		APIKey: os.Getenv("API_KEY"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.DBUrl == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("API_KEY is required")
	}

	return cfg, nil
}
