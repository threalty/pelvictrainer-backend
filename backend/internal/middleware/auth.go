package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pelvictrainer/backend/internal/auth"
)

// AuthMiddleware проверяет JWT токен
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из заголовка Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Отсутствует заголовок Authorization",
			})
			c.Abort()
			return
		}

		// Проверяем формат "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Неверный формат токена",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Валидируем токен
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Недействительный токен",
			})
			c.Abort()
			return
		}

		// Сохраняем данные пользователя в контекст
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role) // НОВОЕ: добавляем роль

		c.Next()
	}
}