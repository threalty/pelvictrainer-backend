package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionHandler struct {
	db *pgxpool.Pool
}

func NewSubscriptionHandler(db *pgxpool.Pool) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

type Subscription struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	Plan      string     `json:"plan"`
	Status    string     `json:"status"`
	StartsAt  time.Time  `json:"starts_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// GetSubscriptions возвращает список всех подписок с информацией о пользователях
func (h *SubscriptionHandler) GetSubscriptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT 
			s.id, s.user_id, s.plan, s.status, s.starts_at, s.expires_at, s.created_at,
			u.email, u.name
		FROM subscriptions s
		JOIN users u ON s.user_id = u.id
		ORDER BY s.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса к БД"})
		return
	}
	defer rows.Close()

	type SubscriptionWithUser struct {
		Subscription
		UserEmail string `json:"user_email"`
		UserName  string `json:"user_name"`
	}

	var subs []SubscriptionWithUser
	for rows.Next() {
		var s SubscriptionWithUser
		err := rows.Scan(
			&s.ID, &s.UserID, &s.Plan, &s.Status, &s.StartsAt, &s.ExpiresAt, &s.CreatedAt,
			&s.UserEmail, &s.UserName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка чтения данных"})
			return
		}
		subs = append(subs, s)
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subs,
		"count":         len(subs),
	})
}

// ActivateSubscription активирует подписку для пользователя
func (h *SubscriptionHandler) ActivateSubscription(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	var req struct {
		Plan       string `json:"plan" binding:"required,oneof=monthly yearly lifetime"`
		DurationDays int  `json:"duration_days"` // для monthly/yearly
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Проверяем что пользователь существует
	var userExists bool
	err = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists)
	if err != nil || !userExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	// Отменяем предыдущие активные подписки
	_, err = h.db.Exec(ctx, `
		UPDATE subscriptions 
		SET status = 'cancelled', updated_at = NOW() 
		WHERE user_id = $1 AND status = 'active'
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отмены старых подписок"})
		return
	}

	// Рассчитываем expires_at
	now := time.Now()
	var expiresAt *time.Time
	switch req.Plan {
	case "monthly":
		t := now.AddDate(0, 1, 0)
		expiresAt = &t
	case "yearly":
		t := now.AddDate(1, 0, 0)
		expiresAt = &t
	case "lifetime":
		expiresAt = nil
	}

	// Создаём новую подписку
	var sub Subscription
	err = h.db.QueryRow(ctx, `
		INSERT INTO subscriptions (user_id, plan, status, starts_at, expires_at)
		VALUES ($1, $2, 'active', $3, $4)
		RETURNING id, user_id, plan, status, starts_at, expires_at, created_at
	`, userID, req.Plan, now, expiresAt).Scan(
		&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.StartsAt, &sub.ExpiresAt, &sub.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания подписки"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Подписка активирована",
		"subscription": sub,
	})
}

// GetUserSubscription возвращает активную подписку конкретного пользователя
func (h *SubscriptionHandler) GetUserSubscription(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var sub Subscription
	err = h.db.QueryRow(ctx, `
		SELECT id, user_id, plan, status, starts_at, expires_at, created_at
		FROM subscriptions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.StartsAt, &sub.ExpiresAt, &sub.CreatedAt)

	if err != nil {
		// Подписки нет — это нормально
		c.JSON(http.StatusOK, gin.H{
			"subscription": nil,
			"status":       "no_subscription",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub,
		"status":       "active",
	})
}

// CancelSubscription отменяет подписку
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	subIDStr := c.Param("id")
	subID, err := strconv.Atoi(subIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := h.db.Exec(ctx, `
		UPDATE subscriptions 
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, subID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка отмены подписки"})
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Активная подписка не найдена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Подписка отменена",
	})
}