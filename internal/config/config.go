package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv           string
	AppPort          string
	DatabaseURL      string
	JWTSecret        string
	GeminiAPIKey     string
	GeminiModel      string
	AITimeoutSeconds int
}

func Load() (Config, error) {
	timeoutSeconds := 10
	if value, err := strconv.Atoi(getEnv("AI_TIMEOUT_SECONDS", "10")); err == nil && value > 0 {
		timeoutSeconds = value
	}
	cfg := Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		AppPort:          getEnv("APP_PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		GeminiAPIKey:     os.Getenv("GEMINI_API_KEY"),
		GeminiModel:      getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		AITimeoutSeconds: timeoutSeconds,
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

func (c Config) Address() string {
	return ":" + c.AppPort
}
