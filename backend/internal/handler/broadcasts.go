package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"pelvictrainer/backend/internal/email"
)

// BroadcastHandler обработчики рассылок
type BroadcastHandler struct {
	db          *pgxpool.Pool
	emailSender *email.Sender
}

// NewBroadcastHandler создаёт обработчик рассылок
func NewBroadcastHandler(db *pgxpool.Pool, emailSender *email.Sender) *BroadcastHandler {
	return &BroadcastHandler{
		db:          db,
		emailSender: emailSender,
	}
}

// Broadcast структура рассылки
type Broadcast struct {
	ID          int        `json:"id"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	Audience    string     `json:"audience"`
	Status      string     `json:"status"`
	SentCount   int        `json:"sent_count"`
	FailedCount int        `json:"failed_count"`
	CreatedAt   time.Time  `json:"created_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
}

// CreateBroadcastRequest запрос на создание рассылки
type CreateBroadcastRequest struct {
	Subject  string `json:"subject" binding:"required"`
	Body     string `json:"body" binding:"required"`
	Audience string `json:"audience" binding:"required,oneof=all premium free inactive"`
}

// AdminCreateBroadcast создаёт новую рассылку (админ)
func (h *BroadcastHandler) AdminCreateBroadcast(c *gin.Context) {
	var req CreateBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var broadcastID int
	err := h.db.QueryRow(ctx, `
		INSERT INTO broadcasts (subject, body, audience, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id
	`, req.Subject, req.Body, req.Audience).Scan(&broadcastID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания рассылки"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Рассылка создана",
		"broadcast_id": broadcastID,
	})
}

// AdminSendBroadcast отправляет рассылку
func (h *BroadcastHandler) AdminSendBroadcast(c *gin.Context) {
	broadcastIDStr := c.Param("id")
	broadcastID, err := strconv.Atoi(broadcastIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Получаем рассылку
	var broadcast Broadcast
	err = h.db.QueryRow(ctx, `
		SELECT id, subject, body, audience, status, sent_count, failed_count, created_at, sent_at
		FROM broadcasts
		WHERE id = $1
	`, broadcastID).Scan(
		&broadcast.ID,
		&broadcast.Subject,
		&broadcast.Body,
		&broadcast.Audience,
		&broadcast.Status,
		&broadcast.SentCount,
		&broadcast.FailedCount,
		&broadcast.CreatedAt,
		&broadcast.SentAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Рассылка не найдена"})
		return
	}

	if broadcast.Status == "sending" || broadcast.Status == "sent" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Рассылка уже отправлена или отправляется"})
		return
	}

	// Помечаем как "отправляется"
	_, _ = h.db.Exec(ctx, `UPDATE broadcasts SET status = 'sending' WHERE id = $1`, broadcastID)

	// Получаем список получателей в зависимости от аудитории
	recipients, err := h.getRecipients(ctx, broadcast.Audience)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения списка получателей"})
		return
	}

	if len(recipients) == 0 {
		_, _ = h.db.Exec(ctx, `UPDATE broadcasts SET status = 'sent', sent_at = NOW() WHERE id = $1`, broadcastID)
		c.JSON(http.StatusOK, gin.H{
			"message":     "Рассылка отправлена",
			"sent_count":  0,
			"failed_count": 0,
		})
		return
	}

	// Отправляем письма в горутине
	go func() {
		sendCtx, sendCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer sendCancel()

		sentCount := 0
		failedCount := 0

		for _, recipient := range recipients {
			msg := email.EmailMessage{
				To:      []string{recipient},
				Subject: broadcast.Subject,
				Body:    broadcast.Body,
				IsHTML:  true,
			}

			if err := h.emailSender.SendMultiple(msg); err != nil {
				failedCount++
				fmt.Printf("⚠️ Ошибка отправки на %s: %v\n", recipient, err)
			} else {
				sentCount++
			}

			// Небольшая задержка между письмами
			time.Sleep(500 * time.Millisecond)
		}

		// Обновляем статистику
		_, _ = h.db.Exec(sendCtx, `
			UPDATE broadcasts 
			SET status = 'sent', sent_count = $1, failed_count = $2, sent_at = NOW()
			WHERE id = $3
		`, sentCount, failedCount, broadcastID)

		fmt.Printf("✅ Рассылка #%d завершена: отправлено=%d, ошибок=%d\n", broadcastID, sentCount, failedCount)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":       "Рассылка отправляется",
		"total_recipients": len(recipients),
	})
}

// AdminGetBroadcasts список всех рассылок (админ)
func (h *BroadcastHandler) AdminGetBroadcasts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, subject, body, audience, status, sent_count, failed_count, created_at, sent_at
		FROM broadcasts
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}
	defer rows.Close()

	broadcasts := make([]Broadcast, 0)
	for rows.Next() {
		var b Broadcast
		if err := rows.Scan(&b.ID, &b.Subject, &b.Body, &b.Audience, &b.Status, &b.SentCount, &b.FailedCount, &b.CreatedAt, &b.SentAt); err == nil {
			broadcasts = append(broadcasts, b)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"broadcasts": broadcasts,
		"count":      len(broadcasts),
	})
}

// getRecipients возвращает список email в зависимости от аудитории
func (h *BroadcastHandler) getRecipients(ctx context.Context, audience string) ([]string, error) {
	var query string

	switch audience {
	case "all":
		query = `SELECT email FROM users`
	case "premium":
		query = `
			SELECT DISTINCT u.email 
			FROM users u
			JOIN subscriptions s ON u.id = s.user_id
			WHERE s.status = 'active'
		`
	case "free":
		query = `
			SELECT DISTINCT u.email 
			FROM users u
			LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status = 'active'
			WHERE s.id IS NULL
		`
	case "inactive":
		query = `
			SELECT DISTINCT u.email 
			FROM users u
			LEFT JOIN training_sessions ts ON u.id = ts.user_id AND ts.completed_at > NOW() - INTERVAL '7 days'
			WHERE ts.id IS NULL
		`
	default:
		query = `SELECT email FROM users`
	}

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]string, 0)
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil {
			emails = append(emails, email)
		}
	}

	return emails, nil
}