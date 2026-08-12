package main

import (
	"log"
	"net/http"

	"pelvictrainer/backend/internal/config"
	"pelvictrainer/backend/internal/handler"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	log.Println("🚀 Запуск PelvicTrainer API...")
	log.Printf("   Порт: %s", cfg.Port)
	log.Printf("   Окружение: %s", cfg.Environment)

	// Создаём роутер
	router := handler.SetupRouter()

	// Запускаем HTTP сервер
	log.Printf("✅ API доступен на http://localhost:%s/health", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
}