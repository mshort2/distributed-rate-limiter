package limiter

import (
	"context"
    "fmt"
	"time"
	"github.com/mshort2/distributed-rate-limiter/pkg/redis"
	"github.com/mshort2/distributed-rate-limiter/internal/config"
)

type RateLimitResponse struct {
    Allowed     bool   `json:"allowed"`
    Remaining   int    `json:"remaining"`
    ResetTime   time.Time  `json:"reset_time"`
	WindowStart time.Time `json:"window_start"`
    ClientID    string `json:"client_id"`
    Region      string `json:"region"`
    RequestID   string `json:"request_id"`
}

type SlidingWindowLimiter struct {
	redisDB *redis.Client
	limit       int
	windowSize  time.Duration
	sha        string
}

func NewRateLimiter(cfg *config.Config, limit int, windowSize time.Duration) (*SlidingWindowLimiter, error) {
    // Lua script for atomic sliding log rate limiting
    lua := `
    local key = KEYS[1]
    local now = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])
    local limit = tonumber(ARGV[3])
    redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
    local count = redis.call("ZCARD", key)
    if count >= limit then
        return {0, limit - count}
    end
    redis.call("ZADD", key, now, now)
    redis.call("EXPIRE", key, math.ceil(window / 1000) * 2)
    return {1, limit - (count + 1)}
    `

	client, err := redis.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	sha, err := client.ScriptLoad(context.Background(), lua)
	if err != nil {
		return &SlidingWindowLimiter{}, fmt.Errorf("failed to load script: %w", err)
	}

    return &SlidingWindowLimiter{
        redisDB:     client,
        limit:      limit,
        windowSize: windowSize,
        sha:       sha,
    }, nil

}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string, requestID string) (RateLimitResponse, error) {
	now := time.Now().UnixMilli()
	window := int64(l.windowSize.Milliseconds())
	limit := int64(l.limit)

	// Execute the Lua script atomically
	result, err := l.redisDB.EvalSha(ctx, l.sha, []string{key}, now, window, limit)
	if err != nil {
		return RateLimitResponse{}, fmt.Errorf("failed to execute rate limit script: %w", err)
	}

	vals, ok := result.([]interface{})
	if !ok || len(vals) != 2 {
		return RateLimitResponse{}, fmt.Errorf("unexpected script result: %#v", result)
	}

	allowed := vals[0].(int64) != 0
	remaining := int(vals[1].(int64))

	resetTime := time.UnixMilli(now + window)
	windowStart := time.UnixMilli(now - window)

	return RateLimitResponse{
		Allowed:     allowed,
		Remaining:   remaining,
		ResetTime:   resetTime,
		WindowStart: windowStart,
		ClientID:    key,
		RequestID:   requestID,
	}, nil
}