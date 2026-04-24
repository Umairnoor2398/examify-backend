package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*ipLimiter)
	limitersMu sync.Mutex
)

func init() {
	go cleanupLimiters()
}

func cleanupLimiters() {
	for range time.Tick(5 * time.Minute) {
		limitersMu.Lock()
		for ip, l := range limiters {
			if time.Since(l.lastSeen) > 10*time.Minute {
				delete(limiters, ip)
			}
		}
		limitersMu.Unlock()
	}
}

func getLimiter(ip string) *rate.Limiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()

	if l, ok := limiters[ip]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}

	// 10 requests per minute with a burst of 20
	l := &ipLimiter{
		limiter:  rate.NewLimiter(rate.Every(6*time.Second), 20),
		lastSeen: time.Now(),
	}
	limiters[ip] = l
	return l.limiter
}

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
