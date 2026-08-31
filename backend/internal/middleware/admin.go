package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware проверяет что пользователь имеет роль admin
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Требуется авторизация",
			})
			c.Abort()
			return
		}

		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Доступ запрещён. Требуются права администратора",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}