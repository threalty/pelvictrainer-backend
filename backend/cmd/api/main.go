package main

import (
	"log"
	"net/http"

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

	// Передаём пул подключений в handler
	handler.SetDBPool(pgDB.Pool)

	// Создаём роутер
	router := handler.SetupRouter()

	// Запускаем HTTP сервер
	log.Printf("✅ API доступен на http://localhost:%s/health", cfg.Port)
	log.Printf("✅ Endpoints:")
	log.Printf("   GET  /health")
	log.Printf("   GET  /ready")
	log.Printf("   GET  /api/v1/ping")
	log.Printf("   GET  /api/v1/users")
	log.Printf("   POST /api/v1/users")
	
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
}