package config

import (
	"log"
	"os"
)

// Config содержит конфигурацию приложения
type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	AppBaseURL  string // базовый URL приложения (для ссылок в письмах)

	// Email (SMTP) настройки
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pelvictrainer?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		AppBaseURL:   getEnv("APP_BASE_URL", "http://localhost:8080"),
		SMTPHost:     getEnv("SMTP_HOST", "smtp.yandex.ru"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@pelvictrainer.ru"),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "PelvicTrainer"),
	}

	if cfg.SMTPUsername == "" {
		log.Println("⚠️ SMTP_USERNAME не задан — отправка email будет недоступна")
	}

	return cfg
}

// getEnv возвращает значение переменной окружения или дефолт
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}