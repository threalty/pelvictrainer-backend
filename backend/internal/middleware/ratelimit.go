package middleware

import (
"net/http"
"sync"
"time"

"github.com/gin-gonic/gin"
)

// rateLimiter простой sliding window rate limiter
type rateLimiter struct {
mu       sync.Mutex
requests map[string][]time.Time
limit    int
window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
rl := &rateLimiter{
requests: make(map[string][]time.Time),
limit:    limit,
window:   window,
}

// Фоновая очистка старых записей
go func() {
ticker := time.NewTicker(5 * time.Minute)
for range ticker.C {
rl.mu.Lock()
cutoff := time.Now().Add(-rl.window)
for key, times := range rl.requests {
var recent []time.Time
for _, t := range times {
if t.After(cutoff) {
recent = append(recent, t)
}
}
if len(recent) == 0 {
delete(rl.requests, key)
} else {
rl.requests[key] = recent
}
}
rl.mu.Unlock()
}
}()

return rl
}

func (rl *rateLimiter) allow(key string) bool {
rl.mu.Lock()
defer rl.mu.Unlock()

now := time.Now()
cutoff := now.Add(-rl.window)

var recent []time.Time
for _, t := range rl.requests[key] {
if t.After(cutoff) {
recent = append(recent, t)
}
}

if len(recent) >= rl.limit {
rl.requests[key] = recent
return false
}

rl.requests[key] = append(recent, now)
return true
}

// RateLimitMiddleware ограничивает запросы по IP
// limit: макс запросов, window: период
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
rl := newRateLimiter(limit, window)

return func(c *gin.Context) {
key := c.ClientIP()
if !rl.allow(key) {
c.JSON(http.StatusTooManyRequests, gin.H{
"error": "Слишком много запросов, попробуйте позже",
})
c.Abort()
return
}
c.Next()
}
}
