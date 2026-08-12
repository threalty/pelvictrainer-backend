package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config хранит все настройки приложения
type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	Environment    string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	// Загружаем .env файл если существует (для локальной разработки)
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://pelvic:ChangeMe_StrongPassword_123!@localhost:5432/pelvictrainer?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://:RedisStrongPass_456!@localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// getEnv возвращает значение переменной окружения или default
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}