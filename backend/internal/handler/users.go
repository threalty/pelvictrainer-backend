package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserHandler обработчики для пользователей
type UserHandler struct {
	db *pgxpool.Pool
}

// NewUserHandler создаёт обработчик пользователей
func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{db: db}
}

// User структура пользователя
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// GetUsers возвращает список всех пользователей (для админки)
func (h *UserHandler) GetUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, email, name, created_at 
		FROM users 
		ORDER BY created_at DESC 
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка запроса к БД",
		})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Ошибка чтения данных",
			})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// CreateUser создаёт нового пользователя (для тестирования)
func (h *UserHandler) CreateUser(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user User
	err := h.db.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash) 
		VALUES ($1, $2, $3) 
		RETURNING id, email, name, created_at
	`, input.Email, input.Name, "placeholder_hash").Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка создания пользователя",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Пользователь создан",
		"user":    user,
	})
}

// GetUserDetail возвращает детальную информацию о пользователе
func (h *UserHandler) GetUserDetail(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Получаем базовую информацию о пользователе
	var user struct {
		ID        int       `json:"id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
	}

	err = h.db.QueryRow(ctx, `
		SELECT id, email, name, created_at 
		FROM users 
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	// Статистика: всего тренировок
	var totalSessions int
	err = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM training_sessions WHERE user_id = $1
	`, userID).Scan(&totalSessions)
	if err != nil {
		totalSessions = 0
	}

	// Статистика: последняя тренировка
	var lastSessionDate *time.Time
	err = h.db.QueryRow(ctx, `
		SELECT MAX(completed_at) FROM training_sessions WHERE user_id = $1
	`, userID).Scan(&lastSessionDate)
	if err != nil {
		lastSessionDate = nil
	}

	// Статистика: streak (непрерывные дни тренировок)
	streak := calculateStreak(ctx, h.db, userID)

	// Активность: дней с последней тренировки
	daysSinceLast := 0
	if lastSessionDate != nil {
		daysSinceLast = int(time.Since(*lastSessionDate).Hours() / 24)
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"stats": gin.H{
			"total_sessions":    totalSessions,
			"current_streak":    streak,
			"days_since_last":   daysSinceLast,
			"last_session_date": lastSessionDate,
		},
	})
}

// GetUserSessions возвращает историю тренировок пользователя
func (h *UserHandler) GetUserSessions(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный user_id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, preset_id, completed_at, duration_seconds, repeats_completed
		FROM training_sessions
		WHERE user_id = $1
		ORDER BY completed_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
		return
	}
	defer rows.Close()

	type Session struct {
		ID               int       `json:"id"`
		PresetID         int       `json:"preset_id"`
		CompletedAt      time.Time `json:"completed_at"`
		DurationSeconds  int       `json:"duration_seconds"`
		RepeatsCompleted int       `json:"repeats_completed"`
	}

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(&s.ID, &s.PresetID, &s.CompletedAt, &s.DurationSeconds, &s.RepeatsCompleted)
		if err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// calculateStreak считает непрерывные дни тренировок
func calculateStreak(ctx context.Context, db *pgxpool.Pool, userID int) int {
	rows, err := db.Query(ctx, `
		SELECT DISTINCT DATE(completed_at) as training_date
		FROM training_sessions
		WHERE user_id = $1
		ORDER BY training_date DESC
	`, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err == nil {
			dates = append(dates, d)
		}
	}

	if len(dates) == 0 {
		return 0
	}

	streak := 1
	today := time.Now().Truncate(24 * time.Hour)
	lastDate := dates[0].Truncate(24 * time.Hour)

	// Если последняя тренировка не сегодня и не вчера — streak = 0
	daysDiff := int(today.Sub(lastDate).Hours() / 24)
	if daysDiff > 1 {
		return 0
	}

	// Считаем непрерывные дни
	for i := 1; i < len(dates); i++ {
		prevDate := dates[i-1].Truncate(24 * time.Hour)
		currDate := dates[i].Truncate(24 * time.Hour)
		diff := int(prevDate.Sub(currDate).Hours() / 24)
		
		if diff == 1 {
			streak++
		} else {
			break
		}
	}

	return streak
}