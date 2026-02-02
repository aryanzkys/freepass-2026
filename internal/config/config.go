package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv      string
	AppPort     string
	DatabaseURL string
	JWTSecret   string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
