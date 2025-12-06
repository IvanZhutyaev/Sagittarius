package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenBucket struct {
	tokens     int
	capacity   int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

func NewRateLimiter(capacity int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     10, // default capacity
			capacity:   10,
			refillRate: time.Second,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / bucket.refillRate)
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.capacity, bucket.tokens+tokensToAdd)
		bucket.lastRefill = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address for rate limiting
		key := c.ClientIP()
		
		// Optionally use user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			key = fmt.Sprintf("user:%s", userID.(string))
		} else {
			key = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		if !limiter.Allow(key) {
			c.JSON(429, gin.H{
				"error": "rate limit exceeded",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

