package handler

import (
"context"
"fmt"
"net/http"
"time"

"github.com/gin-gonic/gin"
"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsHandler struct {
db *pgxpool.Pool
}

func NewAnalyticsHandler(db *pgxpool.Pool) *AnalyticsHandler {
return &AnalyticsHandler{db: db}
}

// Prices для расчёта MRR (в рублях в месяц)
const (
PriceMonthly      = 499
PriceYearly       = 3990
MonthlyFromYearly = float64(PriceYearly) / 12.0 // 332.5
)

// Overview возвращает ключевые метрики
func (h *AnalyticsHandler) Overview(c *gin.Context) {
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

// Всего пользователей
var totalUsers int
h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

// Новых за 7 дней
var newUsers7d int
h.db.QueryRow(ctx, `
SELECT COUNT(*) FROM users 
WHERE created_at > NOW() - INTERVAL '7 days'
`).Scan(&newUsers7d)

// Новых за 30 дней
var newUsers30d int
h.db.QueryRow(ctx, `
SELECT COUNT(*) FROM users 
WHERE created_at > NOW() - INTERVAL '30 days'
`).Scan(&newUsers30d)

// Активных подписок
var activeSubs int
h.db.QueryRow(ctx, `
SELECT COUNT(*) FROM subscriptions 
WHERE status = 'active'
`).Scan(&activeSubs)

// MRR: считаем только активные подписки
var mrrMonthly, mrrYearly int
h.db.QueryRow(ctx, `
SELECT COUNT(*) FROM subscriptions 
WHERE status = 'active' AND plan = 'monthly'
`).Scan(&mrrMonthly)
h.db.QueryRow(ctx, `
SELECT COUNT(*) FROM subscriptions 
WHERE status = 'active' AND plan = 'yearly'
`).Scan(&mrrYearly)

mrr := float64(mrrMonthly)*PriceMonthly + float64(mrrYearly)*MonthlyFromYearly

// Конверсия free → paid
conversionRate := 0.0
if totalUsers > 0 {
conversionRate = float64(activeSubs) / float64(totalUsers) * 100.0
}

// DAU: пользователи с тренировками за последние 24 часа
var dau int
h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT user_id) FROM training_sessions 
WHERE completed_at > NOW() - INTERVAL '24 hours'
`).Scan(&dau)

// WAU
var wau int
h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT user_id) FROM training_sessions 
WHERE completed_at > NOW() - INTERVAL '7 days'
`).Scan(&wau)

c.JSON(http.StatusOK, gin.H{
"total_users":     totalUsers,
"new_users_7d":    newUsers7d,
"new_users_30d":   newUsers30d,
"active_subs":     activeSubs,
"mrr_rub":         mrr,
"conversion_rate": conversionRate,
"dau":             dau,
"wau":             wau,
})
}

// RegistrationsByDay — регистрации по дням за N дней
func (h *AnalyticsHandler) RegistrationsByDay(c *gin.Context) {
daysStr := c.DefaultQuery("days", "30")
var days int
if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil || days < 1 || days > 365 {
days = 30
}

ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

rows, err := h.db.Query(ctx, `
SELECT 
DATE(created_at) as day,
COUNT(*) as count
FROM users
WHERE created_at > NOW() - INTERVAL '1 day' * $1
GROUP BY DATE(created_at)
ORDER BY day ASC
`, days)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
return
}
defer rows.Close()

type DayCount struct {
Date  string `json:"date"`
Count int    `json:"count"`
}

dataMap := make(map[string]int)
for rows.Next() {
var day time.Time
var count int
if err := rows.Scan(&day, &count); err == nil {
dataMap[day.Format("2006-01-02")] = count
}
}

// Заполняем пропуски нулями
result := make([]DayCount, 0, days)
now := time.Now().UTC()
for i := days - 1; i >= 0; i-- {
day := now.AddDate(0, 0, -i)
key := day.Format("2006-01-02")
count := dataMap[key]
result = append(result, DayCount{
Date:  key,
Count: count,
})
}

c.JSON(http.StatusOK, gin.H{
"days": days,
"data": result,
})
}

// SubscriptionBreakdown — распределение подписок
func (h *AnalyticsHandler) SubscriptionBreakdown(c *gin.Context) {
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

// Активные по планам
type PlanCount struct {
Plan  string  `json:"plan"`
Count int     `json:"count"`
MRR   float64 `json:"mrr"`
}

var active []PlanCount
rows, err := h.db.Query(ctx, `
SELECT plan, COUNT(*) as count
FROM subscriptions
WHERE status = 'active'
GROUP BY plan
`)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса"})
return
}
defer rows.Close()

for rows.Next() {
var pc PlanCount
if err := rows.Scan(&pc.Plan, &pc.Count); err == nil {
switch pc.Plan {
case "monthly":
pc.MRR = float64(pc.Count) * PriceMonthly
case "yearly":
pc.MRR = float64(pc.Count) * MonthlyFromYearly
case "lifetime":
pc.MRR = 0 // lifetime не учитываем в MRR
}
active = append(active, pc)
}
}

// Пользователи без подписки (free)
var totalUsers, usersWithActiveSub int
h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)
h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT user_id) FROM subscriptions WHERE status = 'active'
`).Scan(&usersWithActiveSub)
freeCount := totalUsers - usersWithActiveSub

c.JSON(http.StatusOK, gin.H{
"active": active,
"free": gin.H{
"count": freeCount,
"label": "Free",
},
"total_users": totalUsers,
})
}
