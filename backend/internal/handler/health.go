package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/email"
	"pelvictrainer/backend/internal/middleware"
)

var dbPool *pgxpool.Pool

func SetDBPool(pool *pgxpool.Pool) {
	dbPool = pool
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func readinessCheck(db *pgxpool.Pool, redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		// Check DB
		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"db":     "error: " + err.Error(),
			})
			return
		}

		// Check Redis
		if err := redis.Ping(ctx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"redis":  "error: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"db":     "ok",
			"redis":  "ok",
		})
	}
}

func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

func SetupRouter(
	authHandler *AuthHandler,
	jwtService *auth.JWTService,
	emailSender *email.Sender,
	appBaseURL string,
) *gin.Engine {
	router := gin.Default()

	// Health checks (публичные)
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck(dbPool, authHandler.redis))
	router.GET("/api/health", healthCheck)
	router.GET("/api/ready", readinessCheck(dbPool, authHandler.redis))

	// API v1
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", pingHandler)

		// Auth endpoints (публичные)
		authGroup := v1.Group("/auth")
		authGroup.Use(middleware.RateLimitMiddleware(20, time.Minute))
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.RefreshToken)
			authGroup.POST("/logout", authHandler.Logout)

			// Восстановление пароля
			recoveryHandler := NewAuthRecoveryHandler(dbPool, emailSender, appBaseURL)
			authGroup.POST("/forgot-password", recoveryHandler.ForgotPassword)
			authGroup.POST("/reset-password", recoveryHandler.ResetPassword)
			authGroup.POST("/check-token", recoveryHandler.CheckToken)
		}

		// Публичные эндпоинты 2FA (для проверки кода при логине)
		twoFactorAuth := v1.Group("/2fa")
		twoFactorAuth.Use(middleware.RateLimitMiddleware(10, time.Minute))
		{
			// === НОВОЕ: Передаём emailSender в NewTwoAuthHandler ===
			twoFactorHandler := NewTwoAuthHandler(dbPool, jwtService, emailSender)
			twoFactorAuth.POST("/verify-login", twoFactorHandler.VerifyForLogin)
			twoFactorAuth.POST("/verify-backup", twoFactorHandler.VerifyBackupCode)
		}

		// Защищённые эндпоинты (требуют авторизации)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(jwtService))
		{
			// Пользователи (админы)
			userHandler := NewUserHandler(dbPool)
			protected.GET("/users", userHandler.GetUsers)
			protected.GET("/users/:id", userHandler.GetUserDetail)
			protected.GET("/users/:id/sessions", userHandler.GetUserSessions)
			protected.POST("/users", userHandler.CreateUser)

			// Подписки (админы)
			subHandler := NewSubscriptionHandler(dbPool)
			protected.GET("/subscriptions", subHandler.GetSubscriptions)
			protected.GET("/users/:id/subscription", subHandler.GetUserSubscription)
			protected.POST("/users/:id/subscription", subHandler.ActivateSubscription)
			protected.DELETE("/subscriptions/:id", subHandler.CancelSubscription)

			// Аналитика (админы)
			analyticsHandler := NewAnalyticsHandler(dbPool)
			protected.GET("/analytics/overview", analyticsHandler.Overview)
			protected.GET("/analytics/registrations", analyticsHandler.RegistrationsByDay)
			protected.GET("/analytics/subscriptions", analyticsHandler.SubscriptionBreakdown)
			protected.GET("/analytics/cohorts", analyticsHandler.CohortAnalysis)

			// Платежи (админы)
			paymentsHandler := NewPaymentsHandler(dbPool, emailSender)
			protected.GET("/payments", paymentsHandler.AdminGetPayments)
			protected.GET("/revenue", paymentsHandler.AdminGetRevenue)

			// Рассылки (админы)
			broadcastHandler := NewBroadcastHandler(dbPool, emailSender)
			protected.GET("/broadcasts", broadcastHandler.AdminGetBroadcasts)
			protected.POST("/broadcasts", broadcastHandler.AdminCreateBroadcast)
			protected.POST("/broadcasts/:id/send", broadcastHandler.AdminSendBroadcast)

			// 2FA (настройка - защищённые)
			// === НОВОЕ: Передаём emailSender в NewTwoAuthHandler ===
			twoAuthHandler := NewTwoAuthHandler(dbPool, jwtService, emailSender)
			protected.GET("/2fa/status", twoAuthHandler.GetStatus)
			protected.POST("/2fa/setup", twoAuthHandler.GenerateSetup)
			protected.POST("/2fa/verify-setup", twoAuthHandler.VerifySetup)
			protected.POST("/2fa/disable", twoAuthHandler.Disable)
			protected.POST("/2fa/regenerate-backup", twoAuthHandler.RegenerateBackupCodes)

			// === Админские эндпоинты для управления 2FA ===
			adminTwoAuthHandler := NewAdminTwoAuthHandler(dbPool)
			
			// Группа роутов только для админов
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.GET("/users/:id/2fa-status", adminTwoAuthHandler.GetUser2FAStatus)
				admin.POST("/users/:id/disable-2fa", adminTwoAuthHandler.DisableUser2FA)
				admin.GET("/users-with-2fa", adminTwoAuthHandler.ListUsersWith2FAStatus)
			}
		}

		// Мобильное приложение (для обычных пользователей)
		mobile := v1.Group("")
		mobile.Use(middleware.AuthMiddleware(jwtService))
		mobile.Use(middleware.RateLimitMiddleware(120, time.Minute))
		{
			mobileHandler := NewMobileHandler(dbPool)
			mobile.GET("/presets", mobileHandler.GetPresets)
			mobile.POST("/sessions", mobileHandler.LogSession)
			mobile.GET("/me/sessions", mobileHandler.GetMySessions)
			mobile.GET("/me/stats", mobileHandler.GetMyStats)
			mobile.GET("/me/subscription", func(c *gin.Context) {
				subHandler := NewSubscriptionHandler(dbPool)
				subHandler.GetUserSubscription(c)
			})
			mobile.POST("/devices", mobileHandler.RegisterDevice)
			mobile.DELETE("/devices", mobileHandler.UnregisterDevice)

			paymentsHandler := NewPaymentsHandler(dbPool, emailSender)
			mobile.POST("/payments/create", paymentsHandler.CreatePayment)
			mobile.GET("/me/payments", paymentsHandler.GetMyPayments)
		}
	}

	return router
}