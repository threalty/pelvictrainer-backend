package main

import (
	"log"
	"net/http"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/config"
	"pelvictrainer/backend/internal/db"
	"pelvictrainer/backend/internal/email"
	"pelvictrainer/backend/internal/handler"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	log.Println("🚀 Запуск PelvicTrainer API...")
	log.Printf("   Порт: %s", cfg.Port)
	log.Printf("   Окружение: %s", cfg.Environment)
	log.Printf("   App Base URL: %s", cfg.AppBaseURL)

	// Подключаемся к PostgreSQL
	pgDB, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Подключаемся к Redis
	redisClient, err := db.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к Redis: %v", err)
	}
	defer redisClient.Close()

	// Создаём JWT сервис
	jwtService := auth.NewJWTService(cfg.JWTSecret)

	// Передаём пул подключений в handler
	handler.SetDBPool(pgDB.Pool)

	// Инициализируем email-сервис
	var emailSender *email.Sender
	if cfg.SMTPUsername != "" {
		emailSender = email.NewSender(
			cfg.SMTPHost,
			cfg.SMTPPort,
			cfg.SMTPUsername,
			cfg.SMTPPassword,
			cfg.SMTPFrom,
			cfg.SMTPFromName,
		)
		log.Println("✅ Email-сервис инициализирован")
	} else {
		log.Println("⚠️ Email-сервис не инициализирован (нет SMTP-настроек)")
	}

	// Создаём auth handler
	authHandler := handler.NewAuthHandler(pgDB.Pool, redisClient, jwtService)

	// Создаём роутер
	router := handler.SetupRouter(authHandler, jwtService, emailSender, cfg.AppBaseURL)

	// Запускаем HTTP сервер
	log.Printf("✅ API доступен на http://localhost:%s/health", cfg.Port)
	log.Printf("✅ Endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   GET  /ready")
	log.Printf("   POST /api/v1/auth/register")
	log.Printf("   POST /api/v1/auth/login")
	log.Printf("   POST /api/v1/auth/refresh")
	log.Printf("   POST /api/v1/auth/logout")
	log.Printf("   POST /api/v1/auth/forgot-password  (НОВОЕ)")
	log.Printf("   POST /api/v1/auth/reset-password   (НОВОЕ)")
	log.Printf("   POST /api/v1/auth/check-token      (НОВОЕ)")

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
}