package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/email"
)

// AuthRecoveryHandler обработчики восстановления пароля
type AuthRecoveryHandler struct {
	db          *pgxpool.Pool
	emailSender *email.Sender
	appBaseURL  string
}

// NewAuthRecoveryHandler создаёт обработчик восстановления пароля
func NewAuthRecoveryHandler(db *pgxpool.Pool, emailSender *email.Sender, appBaseURL string) *AuthRecoveryHandler {
	return &AuthRecoveryHandler{
		db:          db,
		emailSender: emailSender,
		appBaseURL:  appBaseURL,
	}
}

// ForgotPasswordRequest запрос на сброс пароля
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword отправляет ссылку для сброса пароля на email
func (h *AuthRecoveryHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Ищем пользователя по email
	var userID int
	var userName string
	err := h.db.QueryRow(ctx, `
		SELECT id, COALESCE(name, '') 
		FROM users 
		WHERE email = $1
	`, req.Email).Scan(&userID, &userName)

	if err != nil {
		// Возвращаем тот же ответ, чтобы не раскрывать существование аккаунта
		c.JSON(http.StatusOK, gin.H{
			"message": "Если аккаунт с таким email существует, ссылка для сброса пароля отправлена",
		})
		return
	}

	// Генерируем криптографически безопасный токен
	token, err := generateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка генерации токена",
		})
		return
	}

	// Хэшируем токен для хранения в БД
	tokenHash := hashToken(token)

	// Срок действия токена — 1 час
	expiresAt := time.Now().Add(1 * time.Hour)

	// Удаляем старые неиспользованные токены этого пользователя
	_, _ = h.db.Exec(ctx, `
		DELETE FROM password_reset_tokens 
		WHERE user_id = $1 AND used_at IS NULL
	`, userID)

	// Сохраняем новый токен
	_, err = h.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка сохранения токена",
		})
		return
	}

	// Формируем ссылку для сброса пароля
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", h.appBaseURL, token)

	// Отправляем письмо
	if h.emailSender != nil {
		err = h.emailSender.SendPasswordReset(req.Email, userName, resetLink)
		if err != nil {
			fmt.Printf("⚠️ Ошибка отправки письма на %s: %v\n", req.Email, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Если аккаунт с таким email существует, ссылка для сброса пароля отправлена",
	})
}

// ResetPasswordRequest запрос на сброс пароля по токену
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword сбрасывает пароль по токену
func (h *AuthRecoveryHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Хэшируем полученный токен для поиска в БД
	tokenHash := hashToken(req.Token)

	// Ищем валидный токен
	var tokenID int
	var userID int
	err := h.db.QueryRow(ctx, `
		SELECT id, user_id 
		FROM password_reset_tokens 
		WHERE token = $1 
			AND expires_at > NOW() 
			AND used_at IS NULL
	`, tokenHash).Scan(&tokenID, &userID)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверная или устаревшая ссылка для сброса пароля",
		})
		return
	}

	// === ИСПРАВЛЕНО: используем auth.HashPassword ===
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка хэширования пароля",
		})
		return
	}

	// Обновляем пароль пользователя
	result, err := h.db.Exec(ctx, `
		UPDATE users 
		SET password_hash = $1, updated_at = NOW() 
		WHERE id = $2
	`, passwordHash, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка обновления пароля",
		})
		return
	}

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	// Помечаем токен как использованный
	_, _ = h.db.Exec(ctx, `
		UPDATE password_reset_tokens 
		SET used_at = NOW() 
		WHERE id = $1
	`, tokenID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Пароль успешно изменён",
	})
}

// CheckTokenRequest запрос для проверки токена
type CheckTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// CheckToken проверяет валидность токена без сброса пароля
func (h *AuthRecoveryHandler) CheckToken(c *gin.Context) {
	var req CheckTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный формат данных",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tokenHash := hashToken(req.Token)

	var exists bool
	err := h.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM password_reset_tokens 
			WHERE token = $1 AND expires_at > NOW() AND used_at IS NULL
		)
	`, tokenHash).Scan(&exists)

	if err != nil || !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": "Ссылка недействительна или устарела",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
	})
}

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===

// generateSecureToken генерирует криптографически безопасный токен
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken хэширует токен для хранения в БД
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}