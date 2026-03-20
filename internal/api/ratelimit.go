package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware creates a Redis-based sliding window rate limiter.
// maxRequests per window (e.g., 60 per minute).
func RateLimitMiddleware(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s", ip)

		ctx := c.Request.Context()

		// Increment counter
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// On Redis error, allow the request (fail-open)
			c.Next()
			return
		}

		// Set expiry on first request in window
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		// Set rate limit headers
		remaining := int64(maxRequests) - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if count > int64(maxRequests) {
			ttl, _ := rdb.TTL(ctx, key).Result()
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please try again later",
			})
			return
		}

		c.Next()
	}
}
