package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"pelvictrainer/backend/internal/auth"
)

// AuthHandler обработчики аутентификации
type AuthHandler struct {
	db         *pgxpool.Pool
	redis      *redis.Client
	jwtService *auth.JWTService
}

// NewAuthHandler создаёт обработчик аутентификации
func NewAuthHandler(db *pgxpool.Pool, redis *redis.Client, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		db:         db,
		redis:      redis,
		jwtService: jwtService,
	}
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// Register регистрирует нового пользователя
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Проверяем что email не занят
	var exists bool
	err := h.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, req.Email).Scan(&exists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка проверки email",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Пользователь с таким email уже существует",
		})
		return
	}

	// Хэшируем пароль
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка хэширования пароля",
		})
		return
	}

	// Создаём пользователя
	var userID int
	err = h.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, req.Email, passwordHash, req.Name).Scan(&userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка создания пользователя",
		})
		return
	}

	// Генерируем токены
	accessToken, err := h.jwtService.GenerateAccessToken(userID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка генерации токена",
		})
		return
	}

	refreshToken := h.jwtService.GenerateRefreshToken()

	// Сохраняем refresh токен в Redis (на 7 дней)
	err = h.redis.Set(ctx, "refresh:"+refreshToken, userID, 7*24*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка сохранения refresh токена",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Пользователь зарегистрирован",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":    userID,
			"email": req.Email,
			"name":  req.Name,
		},
	})
}

// LoginRequest запрос на вход
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login выполняет вход пользователя
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Ищем пользователя
	var userID int
	var email, name, passwordHash string
	err := h.db.QueryRow(ctx, `
		SELECT id, email, name, password_hash 
		FROM users 
		WHERE email = $1
	`, req.Email).Scan(&userID, &email, &name, &passwordHash)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Неверный email или пароль",
		})
		return
	}

	// Проверяем пароль
	if !auth.CheckPasswordHash(req.Password, passwordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Неверный email или пароль",
		})
		return
	}

	// Генерируем токены
	accessToken, err := h.jwtService.GenerateAccessToken(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка генерации токена",
		})
		return
	}

	refreshToken := h.jwtService.GenerateRefreshToken()

	// Сохраняем refresh токен в Redis
	err = h.redis.Set(ctx, "refresh:"+refreshToken, userID, 7*24*time.Hour).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка сохранения refresh токена",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Вход выполнен",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":    userID,
			"email": email,
			"name":  name,
		},
	})
}

// RefreshTokenRequest запрос на обновление токена
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken обновляет access токен по refresh токену
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Проверяем refresh токен в Redis
	userIDStr, err := h.redis.Get(ctx, "refresh:"+req.RefreshToken).Result()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Недействительный refresh токен",
		})
		return
	}

	// Получаем данные пользователя
	var userID int
	var email, name string
	err = h.db.QueryRow(ctx, `
		SELECT id, email, name 
		FROM users 
		WHERE id = $1
	`, userIDStr).Scan(&userID, &email, &name)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	// Генерируем новый access токен
	accessToken, err := h.jwtService.GenerateAccessToken(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка генерации токена",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// Logout выходит из системы (удаляет refresh токен)
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Удаляем refresh токен из Redis
	h.redis.Del(ctx, "refresh:"+req.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message": "Выход выполнен",
	})
}