package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminTwoAuthHandler обработчики для управления 2FA пользователей админами
type AdminTwoAuthHandler struct {
	db *pgxpool.Pool
}

// NewAdminTwoAuthHandler создаёт обработчик
func NewAdminTwoAuthHandler(db *pgxpool.Pool) *AdminTwoAuthHandler {
	return &AdminTwoAuthHandler{db: db}
}

// User2FAStatusResponse ответ со статусом 2FA пользователя
type User2FAStatusResponse struct {
	UserID     int        `json:"user_id"`
	Email      string     `json:"email"`
	Enabled    bool       `json:"enabled"`
	Configured bool       `json:"configured"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// GetUser2FAStatus возвращает статус 2FA для пользователя
// GET /api/v1/admin/users/:id/2fa-status
func (h *AdminTwoAuthHandler) GetUser2FAStatus(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Получаем email пользователя
	var email string
	err = h.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	// Получаем статус 2FA
	var enabled bool
	var createdAt, updatedAt time.Time
	err = h.db.QueryRow(ctx, `
		SELECT enabled, created_at, updated_at 
		FROM user_2fa 
		WHERE user_id = $1
	`, userID).Scan(&enabled, &createdAt, &updatedAt)

	response := User2FAStatusResponse{
		UserID:  userID,
		Email:   email,
		Enabled: false,
		Configured: false,
	}

	if err == nil {
		response.Enabled = enabled
		response.Configured = true
		response.CreatedAt = &createdAt
		response.UpdatedAt = &updatedAt
	} else if err != pgx.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DisableUser2FARequest запрос на отключение 2FA
type DisableUser2FARequest struct {
	Reason string `json:"reason" binding:"required"`
}

// DisableUser2FA отключает 2FA для пользователя (админ)
// POST /api/v1/admin/users/:id/disable-2fa
func (h *AdminTwoAuthHandler) DisableUser2FA(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	var req DisableUser2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите причину отключения"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Проверяем что пользователь существует
	var email string
	err = h.db.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if err == pgx.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}

	// Получаем ID текущего админа для логирования
	adminID, _ := GetUserIDFromContext(c)

	// Удаляем запись 2FA для пользователя
	result, err := h.db.Exec(ctx, `DELETE FROM user_2fa WHERE user_id = $1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отключения 2FA"})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "2FA не была настроена для этого пользователя",
		})
		return
	}

	// Логируем действие (в продакшене лучше писать в отдельную таблицу audit_log)
	// Здесь просто выводим в лог сервера
	// Можно добавить: log.Printf("Admin %d disabled 2FA for user %d: %s", adminID, userID, req.Reason)

	c.JSON(http.StatusOK, gin.H{
		"message":    "2FA успешно отключена для пользователя",
		"user_id":    userID,
		"email":      email,
		"disabled_by": adminID,
		"reason":     req.Reason,
	})
}

// ListUsersWith2FAStatus возвращает список пользователей со статусом 2FA
// GET /api/v1/admin/users-with-2fa
func (h *AdminTwoAuthHandler) ListUsersWith2FAStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT 
			u.id, 
			u.email, 
			u.name,
			u.created_at,
			COALESCE(t.enabled, false) as two_fa_enabled,
			t.updated_at as two_fa_updated_at
		FROM users u
		LEFT JOIN user_2fa t ON u.id = t.user_id
		ORDER BY u.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}
	defer rows.Close()

	type UserWith2FA struct {
		ID             int        `json:"id"`
		Email          string     `json:"email"`
		Name           string     `json:"name"`
		CreatedAt      time.Time  `json:"created_at"`
		TwoFAEnabled   bool       `json:"two_fa_enabled"`
		TwoFAUpdatedAt *time.Time `json:"two_fa_updated_at,omitempty"`
	}

	var users []UserWith2FA
	for rows.Next() {
		var user UserWith2FA
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.CreatedAt,
			&user.TwoFAEnabled,
			&user.TwoFAUpdatedAt,
		)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}