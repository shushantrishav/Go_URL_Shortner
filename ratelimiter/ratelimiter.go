package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// RateLimitDuration is the window for rate limiting (2 minutes).
	RateLimitDuration = 2 * time.Minute
	// MaxRequests is the maximum number of requests allowed within the duration (15 URLs).
	MaxRequests = 15
)

// RateLimiter uses Redis to implement a simple sliding window rate limit.
type RateLimiter struct {
	client *redis.Client
	ctx    context.Context
}

// NewRateLimiter creates a new RateLimiter instance.
func NewRateLimiter(rc *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: rc,
		ctx:    context.Background(),
	}
}

// Allow checks if the request is allowed based on the rate limit policy.
// It returns whether the request is allowed, the current count of requests in the window, and an error.
func (rl *RateLimiter) Allow(key string) (bool, int64, error) {
	// Use a sorted set to store timestamps of requests.
	// Key format: "ratelimit:<key>"
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	now := time.Now().UnixNano() // Use nanoseconds for precision as scores

	// 1. Remove old requests (outside the window)
	_, err := rl.client.ZRemRangeByScore(rl.ctx, redisKey, "-inf", strconv.FormatInt(now-int64(RateLimitDuration), 10)).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to clean up old rate limit entries: %w", err)
	}

	// 2. Count current requests within the window
	count, err := rl.client.ZCard(rl.ctx, redisKey).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to count rate limit entries: %w", err)
	}

	// 3. Check if the limit is exceeded
	if count >= MaxRequests {
		return false, count, nil // Limit exceeded, return current count
	}

	// 4. Add the current request timestamp
	pipeline := rl.client.Pipeline()
	pipeline.ZAdd(rl.ctx, redisKey, &redis.Z{Score: float64(now), Member: now}).Err()
	pipeline.Expire(rl.ctx, redisKey, RateLimitDuration).Err() // Set TTL on the key itself
	_, err = pipeline.Exec(rl.ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to add request to rate limit: %w", err)
	}

	// After adding, the count increases by 1
	return true, count + 1, nil // Request allowed, return new count
}
