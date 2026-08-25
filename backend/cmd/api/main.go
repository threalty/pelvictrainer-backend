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
	cfg := config.Load()

	log.Println("🚀 Запуск PelvicTrainer API...")
	log.Printf("   Порт: %s", cfg.Port)
	log.Printf("   Окружение: %s", cfg.Environment)
	log.Printf("   App Base URL: %s", cfg.AppBaseURL)

	pgDB, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	redisClient, err := db.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к Redis: %v", err)
	}
	defer redisClient.Close()

	jwtService := auth.NewJWTService(cfg.JWTSecret)

	handler.SetDBPool(pgDB.Pool)

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

	authHandler := handler.NewAuthHandler(pgDB.Pool, redisClient, jwtService, emailSender)

	router := handler.SetupRouter(authHandler, jwtService, emailSender, cfg.AppBaseURL)


	log.Printf("✅ API доступен на http://0.0.0.0:%s/health", cfg.Port)

	if err := http.ListenAndServe("0.0.0.0:"+cfg.Port, router); err != nil {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
}