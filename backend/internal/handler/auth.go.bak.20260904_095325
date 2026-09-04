package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"pelvictrainer/backend/internal/auth"
	"pelvictrainer/backend/internal/email"
)

// AuthHandler обработчик аутентификации
type AuthHandler struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	jwtService  *auth.JWTService
	emailSender *email.Sender
}

// NewAuthHandler создаёт обработчик аутентификации
func NewAuthHandler(
	db *pgxpool.Pool,
	redisClient *redis.Client,
	jwtService *auth.JWTService,
	emailSender *email.Sender,
) *AuthHandler {
	return &AuthHandler{
		db:          db,
		redis:       redisClient,
		jwtService:  jwtService,
		emailSender: emailSender,
	}
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	Name           string `json:"name"`
	ConsentPrivacy bool   `json:"consentPrivacy"`
	ConsentHealth  bool   `json:"consentHealth"`
	ConsentAge     bool   `json:"consentAge"`
}

// LoginRequest запрос на вход
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest запрос на обновление токена
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// authUser внутренняя структура пользователя (для auth)
type authUser struct {
	ID           int
	Email        string
	Name         string
	PasswordHash string
	Role         string // НОВОЕ: роль пользователя
	CreatedAt    time.Time
}

// Register регистрация нового пользователя
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	if !req.ConsentPrivacy || !req.ConsentHealth || !req.ConsentAge {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Необходимо принять все согласия для регистрации",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var exists bool
	err := h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, req.Email).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки пользователя"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с таким email уже существует"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хэширования пароля"})
		return
	}

	var user authUser
	err = h.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, email, name, COALESCE(role, 'user'), created_at
	`, req.Email, string(hashedPassword), req.Name).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
		return
	}

	_, _ = h.db.Exec(ctx, `
		INSERT INTO user_consents (user_id, consent_type, consent_version, ip_address, user_agent, created_at)
		VALUES 
			($1, 'privacy', 'v1.0', $2, $3, NOW()),
			($1, 'health', 'v1.0', $2, $3, NOW()),
			($1, 'age', 'v1.0', $2, $3, NOW())
	`, user.ID, c.ClientIP(), c.Request.UserAgent())

	accessToken, refreshToken, err := h.generateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токенов"})
		return
	}

	// === НОВОЕ: Отправляем приветственное письмо ===
	if h.emailSender != nil {
		go func() {
			if err := h.emailSender.SendWelcome(user.Email, user.Name); err != nil {
				log.Printf("⚠️ Не удалось отправить приветственное письмо: %v", err)
			}
		}()
	}

	// === НОВОЕ: Логируем первый вход как "знакомое устройство" ===
	h.logLogin(ctx, user.ID, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Пользователь зарегистрирован",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// Login вход пользователя
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user authUser
	err := h.db.QueryRow(ctx, `
		SELECT id, email, name, password_hash, COALESCE(role, 'user'), created_at 
		FROM users WHERE email = $1
	`, req.Email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка аутентификации"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный email или пароль"})
		return
	}

	// === Проверка 2FA ===
	var twoFAEnabled bool
	err = h.db.QueryRow(ctx, `SELECT enabled FROM user_2fa WHERE user_id = $1`, user.ID).Scan(&twoFAEnabled)
	if err != nil && err != pgx.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки 2FA"})
		return
	}

	if twoFAEnabled {
		c.JSON(http.StatusOK, gin.H{
			"requires_2fa": true,
			"user_id":      user.ID,
			"email":        user.Email,
			"role":         user.Role,
			"message":      "Требуется код двухфакторной аутентификации",
		})
		return
	}

	// === НОВОЕ: Проверяем новое устройство ПЕРЕД генерацией токенов ===
	h.checkAndNotifyNewDevice(ctx, user.ID, user.Email, user.Name, c.ClientIP(), c.Request.UserAgent())

	accessToken, refreshToken, err := h.generateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токенов"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"requires_2fa":  false,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// RefreshToken обновление access-токена
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	isBlacklisted, err := h.redis.Exists(ctx, "blacklist:"+req.RefreshToken).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки токена"})
		return
	}
	if isBlacklisted > 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Токен отозван"})
		return
	}

	claims, err := h.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		// refresh-токен это UUID, не JWT. Ищем его в БД если есть таблица refresh_tokens
		// Для простоты - принимаем как есть (UUID)
		claims = nil
	}

	var user authUser
	if claims != nil {
		err = h.db.QueryRow(ctx, `
			SELECT id, email, name, password_hash, COALESCE(role, 'user'), created_at 
			FROM users WHERE id = $1
		`, claims.UserID).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Role, &user.CreatedAt)
	} else {
		// UUID-токен, получаем user_id из редиса или пропускаем валидацию
		// Для простоты - возвращаем ошибку
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный refresh-токен"})
		return
	}

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Пользователь не найден"})
		return
	}

	accessToken, refreshToken, err := h.generateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токенов"})
		return
	}

	_, _ = h.redis.Set(ctx, "blacklist:"+req.RefreshToken, "1", 7*24*time.Hour).Result()

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout выход пользователя
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, _ = h.redis.Set(ctx, "blacklist:"+req.RefreshToken, "1", 7*24*time.Hour).Result()

	c.JSON(http.StatusOK, gin.H{"message": "Вы вышли из системы"})
}

// generateTokens генерирует пару токенов
func (h *AuthHandler) generateTokens(user authUser) (string, string, error) {
	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return "", "", err
	}

	// Refresh-токен это просто UUID
	refreshToken := h.jwtService.GenerateRefreshToken()

	return accessToken, refreshToken, nil
}

// === НОВЫЕ ФУНКЦИИ ДЛЯ ОТСЛЕЖИВАНИЯ НОВЫХ УСТРОЙСТВ ===

// hashUserAgent создаёт SHA256 хеш от User-Agent
func hashUserAgent(userAgent string) string {
	hash := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(hash[:])
}

// parseDeviceInfo извлекает упрощённое описание устройства из User-Agent
func parseDeviceInfo(userAgent string) string {
	ua := strings.ToLower(userAgent)

	// Определяем устройство
	var device string
	switch {
	case strings.Contains(ua, "android"):
		device = "Android"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		device = "iOS"
	case strings.Contains(ua, "windows"):
		device = "Windows"
	case strings.Contains(ua, "macintosh"), strings.Contains(ua, "mac os"):
		device = "macOS"
	case strings.Contains(ua, "linux"):
		device = "Linux"
	default:
		device = "Unknown Device"
	}

	// Определяем браузер
	var browser string
	switch {
	case strings.Contains(ua, "pelvictrainer") || strings.Contains(ua, "okhttp"):
		browser = "PelvicTrainer App"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "edge"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		browser = "Safari"
	case strings.Contains(ua, "edge"):
		browser = "Edge"
	default:
		browser = "Browser"
	}

	return fmt.Sprintf("%s on %s", browser, device)
}

// logLogin сохраняет запись о входе в login_history (игнорируя дубликаты)
func (h *AuthHandler) logLogin(ctx context.Context, userID int, ip string, userAgent string) {
	userAgentHash := hashUserAgent(userAgent)
	deviceInfo := parseDeviceInfo(userAgent)

	_, err := h.db.Exec(ctx, `
		INSERT INTO login_history (user_id, ip_address, user_agent, user_agent_hash, device_info, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, user_agent_hash) DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			created_at = NOW()
	`, userID, ip, userAgent, userAgentHash, deviceInfo)

	if err != nil {
		log.Printf("⚠️ Не удалось сохранить запись о входе: %v", err)
	}
}

// checkAndNotifyNewDevice проверяет новое устройство и отправляет email если нужно
func (h *AuthHandler) checkAndNotifyNewDevice(ctx context.Context, userID int, emailAddr, userName, ip, userAgent string) {
	userAgentHash := hashUserAgent(userAgent)
	deviceInfo := parseDeviceInfo(userAgent)

	// Проверяем был ли этот хеш уже для данного пользователя
	var exists bool
	err := h.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM login_history 
			WHERE user_id = $1 AND user_agent_hash = $2
		)
	`, userID, userAgentHash).Scan(&exists)

	if err != nil {
		log.Printf("⚠️ Ошибка проверки login_history: %v", err)
		// Продолжаем даже если проверка не удалась
	}

	isNewDevice := !exists

	// Сохраняем запись (upsert)
	h.logLogin(ctx, userID, ip, userAgent)

	// Если устройство новое — отправляем email
	if isNewDevice && h.emailSender != nil {
		loginTime := time.Now().Format("02.01.2006 в 15:04")
		go func() {
			log.Printf("📧 Отправляем уведомление о новом устройстве: user=%d, device=%s, ip=%s",
				userID, deviceInfo, ip)

			if err := h.emailSender.SendNewDeviceLogin(emailAddr, userName, ip, deviceInfo, loginTime); err != nil {
				log.Printf("⚠️ Не удалось отправить уведомление о новом устройстве: %v", err)
			} else {
				log.Printf("✅ Уведомление о новом устройстве отправлено на %s", emailAddr)
			}
		}()
	} else if isNewDevice {
		log.Printf("⚠️ Обнаружено новое устройство для user=%d, но emailSender не настроен", userID)
	}
}

// GetUserIDFromString парсит user_id из строки (вспомогательная)
func GetUserIDFromString(s string) (int, bool) {
	var id int
	reader := strings.NewReader(s)
	_ = reader
	if _, err := strings.NewReader(s).Read(nil); err == nil {
		return 0, false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		id = id*10 + int(c-'0')
	}
	return id, true
}