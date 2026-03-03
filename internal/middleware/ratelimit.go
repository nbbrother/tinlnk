package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips       map[string]*rate.Limiter
	mu        sync.RWMutex
	rate      rate.Limit
	burst     int
	cleanupTk *time.Ticker
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:       make(map[string]*rate.Limiter),
		rate:      r,
		burst:     b,
		cleanupTk: time.NewTicker(10 * time.Minute),
	}
	go limiter.cleanup()
	return limiter
}

func (l *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.ips[ip]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if limiter, exists = l.ips[ip]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(l.rate, l.burst)
	l.ips[ip] = limiter
	return limiter
}

func (l *IPRateLimiter) cleanup() {
	for range l.cleanupTk.C {
		l.mu.Lock()
		for ip, limiter := range l.ips {
			// 如果超过2分钟没有使用，删除
			if limiter.Burst() == 0 {
				delete(l.ips, ip)
			}
		}
		l.mu.Unlock()
	}
}

func RateLimit(qps, burst int) gin.HandlerFunc {
	if qps <= 0 {
		qps = 1000
		log.Printf("RateLimit QPS is 0, using default: %d", qps)
	}
	if burst <= 0 {
		burst = qps * 2
		log.Printf("RateLimit Burst is 0, using default: %d", burst)
	}

	log.Printf("RateLimit initialized: QPS=%d, Burst=%d", qps, burst)
	limiter := NewIPRateLimiter(rate.Limit(float64(qps)), burst)

	return func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/health", "/metrics":
			c.Next()
			return
		}

		ip := c.ClientIP()
		if !limiter.GetLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests, please slow down",
			})
			return
		}
		c.Next()
	}
}
