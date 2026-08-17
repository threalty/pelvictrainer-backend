package handler

import (
"context"
"net/http"
"strconv"
"time"

"github.com/gin-gonic/gin"
"github.com/jackc/pgx/v5/pgxpool"
)

// MobileHandler endpoints для мобильного приложения
type MobileHandler struct {
db *pgxpool.Pool
}

func NewMobileHandler(db *pgxpool.Pool) *MobileHandler {
return &MobileHandler{db: db}
}

// getUserID достаёт user_id из контекста (ставится AuthMiddleware)
func getUserID(c *gin.Context) (int, bool) {
userID, exists := c.Get("user_id")
if !exists {
return 0, false
}
id, ok := userID.(int)
return id, ok
}

// ===== PRESETS =====

type Preset struct {
ID              int    `json:"id"`
Name            string `json:"name"`
Description     string `json:"description"`
Difficulty      string `json:"difficulty"`
DurationMinutes int    `json:"duration_minutes"`
ExercisesCount  int    `json:"exercises_count"`
}

// GetPresets список активных программ тренировок
func (h *MobileHandler) GetPresets(c *gin.Context) {
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

rows, err := h.db.Query(ctx, `
SELECT id, name, description, difficulty, duration_minutes, exercises_count
FROM presets
WHERE is_active = true
ORDER BY 
CASE difficulty 
WHEN 'beginner' THEN 1 
WHEN 'intermediate' THEN 2 
ELSE 3 
END,
duration_minutes
`)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки программ"})
return
}
defer rows.Close()

presets := make([]Preset, 0)
for rows.Next() {
var p Preset
if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Difficulty, &p.DurationMinutes, &p.ExercisesCount); err == nil {
presets = append(presets, p)
}
}

c.JSON(http.StatusOK, gin.H{
"presets": presets,
"count":   len(presets),
})
}

// ===== SESSIONS =====

// LogSession логирует завершённую тренировку
func (h *MobileHandler) LogSession(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

var req struct {
PresetID         int `json:"preset_id" binding:"required"`
DurationSeconds  int `json:"duration_seconds" binding:"required,min=1"`
RepeatsCompleted int `json:"repeats_completed" binding:"required,min=1"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные: " + err.Error()})
return
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

var sessionID int
err := h.db.QueryRow(ctx, `
INSERT INTO training_sessions (user_id, preset_id, duration_seconds, repeats_completed)
VALUES ($1, $2, $3, $4)
RETURNING id
`, userID, req.PresetID, req.DurationSeconds, req.RepeatsCompleted).Scan(&sessionID)

if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения тренировки"})
return
}

// Возвращаем актуальный streak — приложение покажет "🔥 3 дня подряд!"
streak := calculateStreak(ctx, h.db, userID)

c.JSON(http.StatusCreated, gin.H{
"message":        "Тренировка сохранена",
"session_id":     sessionID,
"current_streak": streak,
})
}

// GetMySessions моя история тренировок
func (h *MobileHandler) GetMySessions(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
if err != nil || limit < 1 || limit > 200 {
limit = 50
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

rows, err := h.db.Query(ctx, `
SELECT s.id, s.preset_id, s.completed_at, s.duration_seconds, s.repeats_completed, 
       COALESCE(p.name, 'Программа удалена')
FROM training_sessions s
LEFT JOIN presets p ON s.preset_id = p.id
WHERE s.user_id = $1
ORDER BY s.completed_at DESC
LIMIT $2
`, userID, limit)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки истории"})
return
}
defer rows.Close()

type MySession struct {
ID               int       `json:"id"`
PresetID         int       `json:"preset_id"`
PresetName       string    `json:"preset_name"`
CompletedAt      time.Time `json:"completed_at"`
DurationSeconds  int       `json:"duration_seconds"`
RepeatsCompleted int       `json:"repeats_completed"`
}

sessions := make([]MySession, 0)
for rows.Next() {
var s MySession
if err := rows.Scan(&s.ID, &s.PresetID, &s.CompletedAt, &s.DurationSeconds, &s.RepeatsCompleted, &s.PresetName); err == nil {
sessions = append(sessions, s)
}
}

c.JSON(http.StatusOK, gin.H{
"sessions": sessions,
"count":    len(sessions),
})
}

// GetMyStats моя статистика
func (h *MobileHandler) GetMyStats(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

var totalSessions, totalMinutes int
h.db.QueryRow(ctx, `
SELECT COUNT(*), COALESCE(SUM(duration_seconds)/60, 0)
FROM training_sessions WHERE user_id = $1
`, userID).Scan(&totalSessions, &totalMinutes)

streak := calculateStreak(ctx, h.db, userID)

var lastSession *time.Time
h.db.QueryRow(ctx, `
SELECT MAX(completed_at) FROM training_sessions WHERE user_id = $1
`, userID).Scan(&lastSession)

c.JSON(http.StatusOK, gin.H{
"total_sessions":  totalSessions,
"total_minutes":   totalMinutes,
"current_streak":  streak,
"last_session_at": lastSession,
})
}

// GetMySubscription проверка статуса подписки (гейтит платный контент)
func (h *MobileHandler) GetMySubscription(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

var plan, status string
var expiresAt *time.Time
err := h.db.QueryRow(ctx, `
SELECT plan, status, expires_at
FROM subscriptions
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT 1
`, userID).Scan(&plan, &status, &expiresAt)

if err != nil {
c.JSON(http.StatusOK, gin.H{
"has_subscription": false,
"plan":             "free",
})
return
}

c.JSON(http.StatusOK, gin.H{
"has_subscription": true,
"plan":             plan,
"status":           status,
"expires_at":       expiresAt,
})
}

// ===== DEVICES =====

// RegisterDevice регистрация устройства для push-уведомлений
func (h *MobileHandler) RegisterDevice(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

var req struct {
FCMToken   string `json:"fcm_token" binding:"required"`
Platform   string `json:"platform"`
AppVersion string `json:"app_version"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
return
}

if req.Platform == "" {
req.Platform = "android"
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

_, err := h.db.Exec(ctx, `
INSERT INTO devices (user_id, fcm_token, platform, app_version, last_seen)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id, fcm_token) 
DO UPDATE SET last_seen = NOW(), app_version = EXCLUDED.app_version
`, userID, req.FCMToken, req.Platform, req.AppVersion)

if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка регистрации устройства"})
return
}

c.JSON(http.StatusOK, gin.H{
"message": "Устройство зарегистрировано",
})
}

// UnregisterDevice отписка от push-уведомлений
func (h *MobileHandler) UnregisterDevice(c *gin.Context) {
userID, ok := getUserID(c)
if !ok {
c.JSON(http.StatusUnauthorized, gin.H{"error": "Нет user_id"})
return
}

var req struct {
FCMToken string `json:"fcm_token" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
return
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

h.db.Exec(ctx, `DELETE FROM devices WHERE user_id = $1 AND fcm_token = $2`, userID, req.FCMToken)

c.JSON(http.StatusOK, gin.H{
"message": "Устройство отписано",
})
}
