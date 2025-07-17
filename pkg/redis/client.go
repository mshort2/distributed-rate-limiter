package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/mshort2/distributed-rate-limiter/internal/config"
)

type Client struct {
    rdb *redis.Client
}

func NewClient(cfg *config.Config) (*Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
        
        // Connection pool settings
        PoolSize:     20,
        MinIdleConns: 5,
        MaxRetries:   3,
        
        // Timeouts
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })

    // Test connection with a background context
    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }

    return &Client{
        rdb: rdb,
    }, nil
}

func (c *Client) Close() error {
    return c.rdb.Close()
}

func (c *Client) Health(ctx context.Context) error {
    return c.rdb.Ping(ctx).Err()
}

// Get current count for a key
func (c *Client) GetCount(ctx context.Context, key string) (int64, error) {
    val, err := c.rdb.Get(ctx, key).Int64()
    if err == redis.Nil {
        return 0, nil
    }
    return val, err
}

// Increment counter with expiration
func (c *Client) IncrementWithExpiry(ctx context.Context, key string, expiry time.Duration) (int64, error) {
    pipe := c.rdb.Pipeline()
    
    incr := pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, expiry)
    
    _, err := pipe.Exec(ctx)
    if err != nil {
        return 0, err
    }
    
    return incr.Val(), nil
}

// Sliding window operations
func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
    return c.rdb.ZAdd(ctx, key, &redis.Z{
        Score:  score,
        Member: member,
    }).Err()
}

func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max string) (int64, error) {
    return c.rdb.ZRemRangeByScore(ctx, key, min, max).Result()
}

func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
    return c.rdb.ZCard(ctx, key).Result()
}

func (c *Client) ZExpire(ctx context.Context, key string, expiry time.Duration) error {
    return c.rdb.Expire(ctx, key, expiry).Err()
}

// Lua script execution for atomic operations
func (c *Client) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
    return c.rdb.EvalSha(ctx, sha1, keys, args...).Result()
}

func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
    return c.rdb.ScriptLoad(ctx, script).Result()
}