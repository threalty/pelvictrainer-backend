package handler

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRouter создаёт и настраивает роутер Gin
func SetupRouter() *gin.Engine {
	// В production режиме отключаем debug логи
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware (разрешаем запросы с admin.pelvictrainer.ru)
	router.Use(corsMiddleware())

	// Health check эндпоинты
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// API версии
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", pingHandler)
	}

	return router
}

// healthCheck - базовая проверка что сервер жив
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "pelvictrainer-api",
		"version": "0.1.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// readinessCheck - проверка что сервер готов принимать запросы
func readinessCheck(c *gin.Context) {
	// TODO: добавить проверку подключения к БД и Redis
	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"service":   "pelvictrainer-api",
		"go_version": runtime.Version(),
	})
}

// pingHandler - простой тестовый эндпоинт
func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong from PelvicTrainer API",
	})
}

// corsMiddleware настраивает CORS для admin UI
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}