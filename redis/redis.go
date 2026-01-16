package redis

import (
    "context"
    "crypto/tls"
    "fmt"

    "github.com/go-redis/redis/v8"
)

// RedisClient represents our Redis client wrapper.
type RedisClient struct {
    client *redis.Client
    ctx    context.Context
}

// NewRedisClient initializes and returns a new RedisClient, now accepting a TLS config.
func NewRedisClient(addr, password string, db int, tlsConfig *tls.Config) (*RedisClient, error) {
    ctx := context.Background()
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
        TLSConfig: tlsConfig,
    })

    // Ping the Redis server to ensure connection is established
    _, err := client.Ping(ctx).Result()
    if err != nil {
        return nil, fmt.Errorf("could not connect to Redis: %w", err)
    }
    fmt.Println("Connected to Redis!")
    return &RedisClient{client: client, ctx: ctx}, nil
}

// Client returns the underlying redis.Client for direct access (used by shortener)
func (r *RedisClient) Client() *redis.Client {
    return r.client
}

// Context returns the context for Redis operations
func (r *RedisClient) Context() context.Context {
    return r.ctx
}

// GetLongURL retrieves the long URL associated with a short URL.
func (r *RedisClient) GetLongURL(shortURL string) (string, error) {
    val, err := r.client.Get(r.ctx, "short:"+shortURL).Result()
    if err == redis.Nil {
        return "", fmt.Errorf("short URL not found")
    } else if err != nil {
        return "", fmt.Errorf("failed to get long URL from Redis: %w", err)
    }
    return val, nil
}

// KeyExists checks if a short slug key exists in Redis.
func (r *RedisClient) KeyExists(key string) (bool, error) {
    count, err := r.client.Exists(r.ctx, "short:"+key).Result()
    if err != nil {
        return false, fmt.Errorf("failed to check key existence: %w", err)
    }
    return count > 0, nil
}
