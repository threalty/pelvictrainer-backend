package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/middleware"
)

var dbPool *pgxpool.Pool

// SetDBPool устанавливает пул подключений к БД
func SetDBPool(pool *pgxpool.Pool) {
	dbPool = pool
}

// SetupRouter создаёт и настраивает роутер Gin
func SetupRouter(authHandler *AuthHandler, jwtService *auth.JWTService) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", pingHandler)

		// Аутентификация (публичные)
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.POST("/logout", authHandler.Logout)
		}

		// Защищённые endpoints
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtService))
		{
			// Пользователи
			userHandler := NewUserHandler(dbPool)
			protected.GET("/users", userHandler.GetUsers)
			protected.POST("/users", userHandler.CreateUser)

			// Подписки (новое!)
			subHandler := NewSubscriptionHandler(dbPool)
			protected.GET("/subscriptions", subHandler.GetSubscriptions)
			protected.GET("/users/:user_id/subscription", subHandler.GetUserSubscription)
			protected.POST("/users/:user_id/subscription", subHandler.ActivateSubscription)
			protected.DELETE("/subscriptions/:id", subHandler.CancelSubscription)
		}
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
	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "БД не подключена",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := dbPool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "БД недоступна",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ready",
		"service":    "pelvictrainer-api",
		"go_version": runtime.Version(),
		"db_status":  "connected",
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