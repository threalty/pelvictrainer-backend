package main

import (
	"log"
	"net/http"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/config"
	"pelvictrainer/backend/internal/db"
	"pelvictrainer/backend/internal/handler"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	log.Println("🚀 Запуск PelvicTrainer API...")
	log.Printf("   Порт: %s", cfg.Port)
	log.Printf("   Окружение: %s", cfg.Environment)

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

	// Создаём auth handler
	authHandler := handler.NewAuthHandler(pgDB.Pool, redisClient, jwtService)

	// Создаём роутер
	router := handler.SetupRouter(authHandler, jwtService)

	// Запускаем HTTP сервер
	log.Printf("✅ API доступен на http://localhost:%s/health", cfg.Port)
	log.Printf("✅ Endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   GET  /ready")
	log.Printf("   POST /api/v1/auth/register")
	log.Printf("   POST /api/v1/auth/login")
	log.Printf("   POST /api/v1/auth/refresh")
	log.Printf("   POST /api/v1/auth/logout")
	log.Printf("   GET  /api/v1/users (protected)")
	log.Printf("   POST /api/v1/users (protected)")

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
}