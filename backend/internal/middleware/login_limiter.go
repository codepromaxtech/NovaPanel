package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// LoginLimiter provides progressive brute-force protection for login endpoints.
// Failed attempts per IP: 5→1min lockout, 10→5min, 15→15min, 20+→1hr.
type LoginLimiter struct {
	rdb *redis.Client
}

func NewLoginLimiter(rdb *redis.Client) *LoginLimiter {
	return &LoginLimiter{rdb: rdb}
}

// Middleware returns a Gin middleware that blocks requests if the IP has too many failed login attempts
func (l *LoginLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("login_fail:%s", ip)
		ctx := context.Background()

		// Check current fail count
		count, err := l.rdb.Get(ctx, key).Int64()
		if err != nil && err != redis.Nil {
			// Redis error — don't block the request
			c.Next()
			return
		}

		if count >= 5 {
			lockDuration := l.getLockDuration(count)

			// Check if currently locked
			lockKey := fmt.Sprintf("login_lock:%s", ip)
			ttl, _ := l.rdb.TTL(ctx, lockKey).Result()
			if ttl > 0 {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":       "Too many failed login attempts. Please try again later.",
					"retry_after": int(ttl.Seconds()),
					"lockout_sec": int(lockDuration.Seconds()),
				})
				return
			}
		}

		c.Next()
	}
}

// RecordFailure increments the failed login counter for an IP and sets a lockout if needed
func (l *LoginLimiter) RecordFailure(ip string) {
	ctx := context.Background()
	key := fmt.Sprintf("login_fail:%s", ip)

	count, _ := l.rdb.Incr(ctx, key).Result()

	// Set expiry on first failure (auto-cleanup after 2 hours of no activity)
	if count == 1 {
		l.rdb.Expire(ctx, key, 2*time.Hour)
	}

	// Apply progressive lockout
	if count >= 5 {
		lockDuration := l.getLockDuration(count)
		lockKey := fmt.Sprintf("login_lock:%s", ip)
		l.rdb.Set(ctx, lockKey, "1", lockDuration)
	}
}

// RecordSuccess resets the failed login counter on successful login
func (l *LoginLimiter) RecordSuccess(ip string) {
	ctx := context.Background()
	l.rdb.Del(ctx, fmt.Sprintf("login_fail:%s", ip))
	l.rdb.Del(ctx, fmt.Sprintf("login_lock:%s", ip))
}

// getLockDuration returns progressive lockout duration based on failure count
func (l *LoginLimiter) getLockDuration(failCount int64) time.Duration {
	switch {
	case failCount >= 20:
		return 1 * time.Hour
	case failCount >= 15:
		return 15 * time.Minute
	case failCount >= 10:
		return 5 * time.Minute
	case failCount >= 5:
		return 1 * time.Minute
	default:
		return 0
	}
}
