package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentsHandler struct {
	db *pgxpool.Pool
}

func NewPaymentsHandler(db *pgxpool.Pool) *PaymentsHandler {
	return &PaymentsHandler{db: db}
}

type Payment struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	AmountCents int        `json:"amount_cents"`
	Currency    string     `json:"currency"`
	Plan        string     `json:"plan"`
	Status      string     `json:"status"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AdminPayment struct {
	Payment
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

type RevenueStats struct {
	TotalRevenueCents   int64 `json:"total_revenue_cents"`
	MonthRevenueCents   int64 `json:"month_revenue_cents"`
	WeekRevenueCents    int64 `json:"week_revenue_cents"`
	TotalPayments       int   `json:"total_payments"`
	ActiveSubscriptions int   `json:"active_subscriptions"`
}

func planAmountCents(plan string) int {
	switch plan {
	case "monthly":
		return 29900
	case "yearly":
		return 249000
	case "lifetime":
		return 499000
	default:
		return 0
	}
}

// ===== MOBILE ENDPOINTS =====

// CreatePayment создание платежа из моб приложения
// Dev-режим: сразу активируем подписку без ЮKassa
func (h *PaymentsHandler) CreatePayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
		return
	}

	var req struct {
		Plan string `json:"plan" binding:"required,oneof=monthly yearly lifetime"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный план: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	amountCents := planAmountCents(req.Plan)
	description := "Подписка PelvicTrainer " + req.Plan
	now := time.Now()

	// Создаём запись о платеже
	var paymentID int
	err := h.db.QueryRow(ctx, `
		INSERT INTO payments (user_id, amount_cents, currency, plan, status, description, created_at, updated_at)
		VALUES ($1, $2, 'RUB', $3, 'succeeded', $4, $5, $5)
		RETURNING id
	`, userID, amountCents, req.Plan, description, now).Scan(&paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания платежа: " + err.Error()})
		return
	}

	// Отменяем предыдущие активные подписки пользователя
	_, _ = h.db.Exec(ctx, `
		UPDATE subscriptions
		SET status = 'cancelled', updated_at = NOW()
		WHERE user_id = $1 AND status = 'active'
	`, userID)

	// Рассчитываем expires_at
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

	// Активируем подписку (UPSERT)
	_, err = h.db.Exec(ctx, `
		INSERT INTO subscriptions (user_id, plan, status, starts_at, expires_at, activated_by_admin, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, $4, false, $5, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET plan = $2, status = 'active', starts_at = $3, expires_at = $4, activated_by_admin = false, updated_at = $5
	`, userID, req.Plan, now, expiresAt, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка активации подписки: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"payment_id": paymentID,
		"status":     "succeeded",
		"message":    "Подписка активирована",
	})
}

// GetMyPayments история платежей текущего пользователя
func (h *PaymentsHandler) GetMyPayments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, user_id, amount_cents, currency, plan, status, description, created_at
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки"})
		return
	}
	defer rows.Close()

	payments := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.UserID, &p.AmountCents, &p.Currency, &p.Plan, &p.Status, &p.Description, &p.CreatedAt); err == nil {
			payments = append(payments, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"count":    len(payments),
	})
}

// ===== ADMIN ENDPOINTS =====

// AdminGetPayments список всех платежей для админки
func (h *PaymentsHandler) AdminGetPayments(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT p.id, p.user_id, p.amount_cents, p.currency, p.plan, p.status,
		       p.description, p.created_at,
		       u.email, u.name
		FROM payments p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}
	defer rows.Close()

	payments := make([]AdminPayment, 0)
	for rows.Next() {
		var p AdminPayment
		if err := rows.Scan(&p.ID, &p.UserID, &p.AmountCents, &p.Currency, &p.Plan, &p.Status, &p.Description, &p.CreatedAt, &p.UserEmail, &p.UserName); err == nil {
			payments = append(payments, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": payments,
		"count":    len(payments),
	})
}

// AdminGetRevenue статистика выручки для админки
func (h *PaymentsHandler) AdminGetRevenue(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var stats RevenueStats

	h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0) FROM payments WHERE status = 'succeeded'
	`).Scan(&stats.TotalRevenueCents)

	h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0) FROM payments
		WHERE status = 'succeeded' AND created_at >= NOW() - INTERVAL '30 days'
	`).Scan(&stats.MonthRevenueCents)

	h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_cents), 0) FROM payments
		WHERE status = 'succeeded' AND created_at >= NOW() - INTERVAL '7 days'
	`).Scan(&stats.WeekRevenueCents)

	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM payments WHERE status = 'succeeded'
	`).Scan(&stats.TotalPayments)

	h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM subscriptions
		WHERE status = 'active' AND (expires_at IS NULL OR expires_at > NOW())
	`).Scan(&stats.ActiveSubscriptions)

	c.JSON(http.StatusOK, stats)
}