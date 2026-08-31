package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/twoauth"
)

// TwoAuthHandler обработчики для двухфакторной аутентификации
type TwoAuthHandler struct {
	db         *pgxpool.Pool
	jwtService *auth.JWTService
}

// NewTwoAuthHandler создаёт обработчик 2FA
func NewTwoAuthHandler(db *pgxpool.Pool, jwtService *auth.JWTService) *TwoAuthHandler {
	return &TwoAuthHandler{db: db, jwtService: jwtService}
}

// SetupResponse ответ на запрос настройки 2FA
type SetupResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
}

// StatusResponse ответ со статусом 2FA
type StatusResponse struct {
	Enabled bool `json:"enabled"`
}

// VerifySetupRequest запрос на подтверждение кода при настройке
type VerifySetupRequest struct {
	Secret string `json:"secret" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// VerifyLoginRequest запрос на проверку кода при логине
type VerifyLoginRequest struct {
	UserID int    `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// VerifyBackupRequest запрос на использование backup-кода
type VerifyBackupRequest struct {
	UserID int    `json:"user_id" binding:"required"`
	Code   string `json:"code" binding:"required"`
}

// DisableRequest запрос на отключение 2FA
type DisableRequest struct {
	Code string `json:"code" binding:"required"`
}

// SetupCompleteResponse содержит backup-коды
type SetupCompleteResponse struct {
	Message     string   `json:"message"`
	BackupCodes []string `json:"backup_codes"`
}

// GenerateSetup генерирует секрет и QR-код
// POST /api/v1/2fa/setup
func (h *TwoAuthHandler) GenerateSetup(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var email string
	err := h.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователя"})
		return
	}

	var enabled bool
	err = h.db.QueryRow(ctx, `SELECT enabled FROM user_2fa WHERE user_id = $1`, userID).Scan(&enabled)
	if err == nil && enabled {
		c.JSON(http.StatusConflict, gin.H{
			"error": "2FA уже включена. Сначала отключите текущую.",
		})
		return
	}

	secret, err := twoauth.GenerateSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации секрета"})
		return
	}

	qrURL := twoauth.GenerateOTPAuthURL(secret, email)

	_, err = h.db.Exec(ctx, `
		INSERT INTO user_2fa (user_id, secret, enabled, created_at, updated_at)
		VALUES ($1, $2, false, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE 
		SET secret = $2, enabled = false, updated_at = NOW()
	`, userID, secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения секрета"})
		return
	}

	c.JSON(http.StatusOK, SetupResponse{
		Secret:    secret,
		QRCodeURL: qrURL,
	})
}

// VerifySetup подтверждает код и активирует 2FA
// POST /api/v1/2fa/verify-setup
func (h *TwoAuthHandler) VerifySetup(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req VerifySetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var storedSecret string
	err := h.db.QueryRow(ctx, `SELECT secret FROM user_2fa WHERE user_id = $1`, userID).Scan(&storedSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Сначала выполните /2fa/setup"})
		return
	}

	if storedSecret != req.Secret {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный секрет"})
		return
	}

	if !twoauth.ValidateCodeWithWindow(storedSecret, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный код"})
		return
	}

	backupCodes, err := twoauth.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации backup-кодов"})
		return
	}

	hashedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hashedCodes[i] = hashBackupCode(code)
	}

	_, err = h.db.Exec(ctx, `
		UPDATE user_2fa 
		SET enabled = true, backup_codes = $2, updated_at = NOW()
		WHERE user_id = $1
	`, userID, hashedCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка активации 2FA"})
		return
	}

	c.JSON(http.StatusOK, SetupCompleteResponse{
		Message:     "2FA успешно активирована. Сохраните backup-коды в надёжном месте!",
		BackupCodes: backupCodes,
	})
}

// GetStatus возвращает статус 2FA пользователя
// GET /api/v1/2fa/status
func (h *TwoAuthHandler) GetStatus(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var enabled bool
	err := h.db.QueryRow(ctx, `SELECT enabled FROM user_2fa WHERE user_id = $1`, userID).Scan(&enabled)
	if err == pgx.ErrNoRows {
		enabled = false
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	c.JSON(http.StatusOK, StatusResponse{Enabled: enabled})
}

// Disable отключает 2FA
// POST /api/v1/2fa/disable
func (h *TwoAuthHandler) Disable(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req DisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var secret string
	err := h.db.QueryRow(ctx, `SELECT secret FROM user_2fa WHERE user_id = $1 AND enabled = true`, userID).Scan(&secret)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA не включена"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	if !twoauth.ValidateCodeWithWindow(secret, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный код"})
		return
	}

	_, err = h.db.Exec(ctx, `DELETE FROM user_2fa WHERE user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отключения 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA отключена"})
}

// VerifyForLogin проверяет код при логине и выдаёт токены
// POST /api/v1/2fa/verify-login
func (h *TwoAuthHandler) VerifyForLogin(c *gin.Context) {
	var req VerifyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var secret string
	err := h.db.QueryRow(ctx, `SELECT secret FROM user_2fa WHERE user_id = $1 AND enabled = true`, req.UserID).Scan(&secret)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA не включена"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	if !twoauth.ValidateCodeWithWindow(secret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный код"})
		return
	}

	// Получаем данные пользователя включая роль
	var email, name, role string
	err = h.db.QueryRow(ctx, `SELECT email, COALESCE(name, ''), COALESCE(role, 'user') FROM users WHERE id = $1`, req.UserID).Scan(&email, &name, &role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Пользователь не найден"})
		return
	}

	// Генерируем access-токен с ролью
	accessToken, err := h.jwtService.GenerateAccessToken(req.UserID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	// Генерируем refresh-токен (UUID)
	refreshToken := h.jwtService.GenerateRefreshToken()

	c.JSON(http.StatusOK, gin.H{
		"message":       "2FA пройдена",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user_id":       req.UserID,
		"email":         email,
		"name":          name,
		"role":          role,
		"authenticated": true,
	})
}

// VerifyBackupCode использует backup-код и выдаёт токены
// POST /api/v1/2fa/verify-backup
func (h *TwoAuthHandler) VerifyBackupCode(c *gin.Context) {
	var req VerifyBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var backupCodes []string
	err := h.db.QueryRow(ctx, `
		SELECT backup_codes FROM user_2fa 
		WHERE user_id = $1 AND enabled = true
	`, req.UserID).Scan(&backupCodes)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA не включена"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	normalizedCode := twoauth.NormalizeBackupCode(req.Code)
	hashedInput := hashBackupCode(normalizedCode)

	found := false
	var remainingCodes []string
	for _, storedCode := range backupCodes {
		if storedCode == hashedInput {
			found = true
		} else {
			remainingCodes = append(remainingCodes, storedCode)
		}
	}

	if !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный backup-код"})
		return
	}

	// Обновляем список (удаляем использованный)
	_, err = h.db.Exec(ctx, `
		UPDATE user_2fa 
		SET backup_codes = $2, updated_at = NOW()
		WHERE user_id = $1
	`, req.UserID, remainingCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления кодов"})
		return
	}

	// Получаем данные пользователя включая роль
	var email, name, role string
	err = h.db.QueryRow(ctx, `SELECT email, COALESCE(name, ''), COALESCE(role, 'user') FROM users WHERE id = $1`, req.UserID).Scan(&email, &name, &role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Пользователь не найден"})
		return
	}

	// Генерируем токены с ролью
	accessToken, err := h.jwtService.GenerateAccessToken(req.UserID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}
	refreshToken := h.jwtService.GenerateRefreshToken()

	c.JSON(http.StatusOK, gin.H{
		"message":                "Backup-код использован",
		"access_token":           accessToken,
		"refresh_token":          refreshToken,
		"user_id":                req.UserID,
		"email":                  email,
		"name":                   name,
		"role":                   role,
		"remaining_backup_codes": len(remainingCodes),
		"authenticated":          true,
	})
}

// RegenerateBackupCodes генерирует новые backup-коды
// POST /api/v1/2fa/regenerate-backup
func (h *TwoAuthHandler) RegenerateBackupCodes(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Требуется авторизация"})
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var secret string
	err := h.db.QueryRow(ctx, `SELECT secret FROM user_2fa WHERE user_id = $1 AND enabled = true`, userID).Scan(&secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA не включена"})
		return
	}

	if !twoauth.ValidateCodeWithWindow(secret, req.Code) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный код"})
		return
	}

	backupCodes, err := twoauth.GenerateBackupCodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации кодов"})
		return
	}

	hashedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		hashedCodes[i] = hashBackupCode(code)
	}

	_, err = h.db.Exec(ctx, `
		UPDATE user_2fa 
		SET backup_codes = $2, updated_at = NOW()
		WHERE user_id = $1
	`, userID, hashedCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения кодов"})
		return
	}

	c.JSON(http.StatusOK, SetupCompleteResponse{
		Message:     "Backup-коды обновлены",
		BackupCodes: backupCodes,
	})
}

// hashBackupCode хэширует backup-код через SHA-256
func hashBackupCode(code string) string {
	hash := sha256.Sum256([]byte(twoauth.NormalizeBackupCode(code)))
	return hex.EncodeToString(hash[:])
}

// GetUserIDFromContext извлекает user_id из gin-контекста
func GetUserIDFromContext(c *gin.Context) (int, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	switch v := userIDInterface.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		var id int
		if _, err := fmt.Sscanf(v, "%d", &id); err == nil {
			return id, true
		}
		return 0, false
	default:
		return 0, false
	}
}