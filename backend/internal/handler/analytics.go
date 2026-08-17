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

// ===== COHORT ANALYSIS =====

type CohortWeek struct {
Week    int     `json:"week"`
Active  int     `json:"active"`
Percent float64 `json:"percent"`
}

type CohortData struct {
CohortWeek string       `json:"cohort_week"`
CohortSize int          `json:"cohort_size"`
Weeks      []CohortWeek `json:"weeks"`
}

type Retention struct {
D1  float64 `json:"d1"`
D7  float64 `json:"d7"`
D30 float64 `json:"d30"`
}

// CohortAnalysis возвращает данные для heatmap когорт
func (h *AnalyticsHandler) CohortAnalysis(c *gin.Context) {
ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
defer cancel()

// Когорты: неделя регистрации × неделя активности
rows, err := h.db.Query(ctx, `
WITH user_cohorts AS (
SELECT 
id AS user_id,
DATE_TRUNC('week', created_at) AS cohort_week
FROM users
),
activity AS (
SELECT DISTINCT
user_id,
DATE_TRUNC('week', completed_at) AS activity_week
FROM training_sessions
)
SELECT 
c.cohort_week,
COUNT(DISTINCT c.user_id) AS cohort_size,
COALESCE(
FLOOR(EXTRACT(EPOCH FROM (a.activity_week - c.cohort_week)) / (7*24*3600)),
-1
) AS week_offset,
COUNT(DISTINCT a.user_id) AS active_users
FROM user_cohorts c
LEFT JOIN activity a ON c.user_id = a.user_id
GROUP BY c.cohort_week, week_offset
ORDER BY c.cohort_week
`)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка запроса когорт"})
return
}
defer rows.Close()

const maxWeeks = 8 // показываем 8 недель

cohortsMap := make(map[string]*CohortData)
var cohortOrder []string

for rows.Next() {
var cohortWeek time.Time
var cohortSize int
var weekOffset int
var activeUsers int

if err := rows.Scan(&cohortWeek, &cohortSize, &weekOffset, &activeUsers); err != nil {
continue
}

key := cohortWeek.Format("2006-01-02")
cohort, exists := cohortsMap[key]
if !exists {
cohort = &CohortData{
CohortWeek: key,
CohortSize: cohortSize,
Weeks:      make([]CohortWeek, maxWeeks),
}
for i := range cohort.Weeks {
cohort.Weeks[i] = CohortWeek{Week: i, Active: 0, Percent: 0}
}
cohortsMap[key] = cohort
cohortOrder = append(cohortOrder, key)
}

// week_offset = -1 означает "нет активности" (LEFT JOIN)
if weekOffset >= 0 && weekOffset < maxWeeks {
percent := 0.0
if cohortSize > 0 {
percent = float64(activeUsers) / float64(cohortSize) * 100.0
}
cohort.Weeks[weekOffset] = CohortWeek{
Week:    int(weekOffset),
Active:  activeUsers,
Percent: percent,
}
}
}

// Собираем в порядке регистрации (последние 12 когорт)
cohorts := make([]CohortData, 0)
start := 0
if len(cohortOrder) > 12 {
start = len(cohortOrder) - 12
}
for i := start; i < len(cohortOrder); i++ {
cohorts = append(cohorts, *cohortsMap[cohortOrder[i]])
}

// ===== Retention D1 / D7 / D30 =====
var totalUsers int
h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&totalUsers)

retention := Retention{}
if totalUsers > 0 {
var d1, d7, d30 int

h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT u.id)
FROM users u
JOIN training_sessions t ON t.user_id = u.id
WHERE t.completed_at <= u.created_at + INTERVAL '1 day'
`).Scan(&d1)

h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT u.id)
FROM users u
JOIN training_sessions t ON t.user_id = u.id
WHERE t.completed_at <= u.created_at + INTERVAL '7 days'
`).Scan(&d7)

h.db.QueryRow(ctx, `
SELECT COUNT(DISTINCT u.id)
FROM users u
JOIN training_sessions t ON t.user_id = u.id
WHERE t.completed_at <= u.created_at + INTERVAL '30 days'
`).Scan(&d30)

retention.D1 = float64(d1) / float64(totalUsers) * 100.0
retention.D7 = float64(d7) / float64(totalUsers) * 100.0
retention.D30 = float64(d30) / float64(totalUsers) * 100.0
}

c.JSON(http.StatusOK, gin.H{
"cohorts":   cohorts,
"retention": retention,
})
}
