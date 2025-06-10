package redis

import (
	"context"
	"crypto/tls" // Import the tls package
	"fmt"
	"time"

	"github.com/go-redis/redis/v8" // Using V8 of go-redis
)

// RedisClient represents our Redis client wrapper.
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient initializes and returns a new RedisClient, now accepting a TLS config.
func NewRedisClient(addr, password string, db int, tlsConfig *tls.Config) (*RedisClient, error) { // Added tlsConfig parameter
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:      addr,
		Password:  password,
		DB:        db,
		TLSConfig: tlsConfig, // Apply the TLS configuration here
	})

	// Ping the Redis server to ensure connection is established
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("could not connect to Redis: %w", err)
	}
	fmt.Println("Connected to Redis!")
	return &RedisClient{client: client, ctx: ctx}, nil
}

// SetURL stores a mapping from short URL to long URL with a TTL.
// It also stores a mapping from long URL to short URL to handle repeated link shortening.
func (r *RedisClient) SetURL(shortURL, longURL string, ttl time.Duration) error {
	// Store shortURL -> longURL mapping
	err := r.client.Set(r.ctx, "short:"+shortURL, longURL, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set short URL in Redis: %w", err)
	}

	// Store longURL -> shortURL mapping for checking duplicates
	// We'll use a specific prefix to differentiate from short:key
	err = r.client.Set(r.ctx, "long:"+longURL, shortURL, ttl).Err()
	if err != nil {
		// If this fails, consider rolling back the previous set, or log it.
		// For now, we'll just log and continue, as the primary shortener still works.
		fmt.Printf("Warning: failed to set long URL mapping for %s: %v\n", longURL, err)
	}
	return nil
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

// GetShortURLByLongURL retrieves the short URL associated with a long URL.
// This is used to check if a long URL has already been shortened.
func (r *RedisClient) GetShortURLByLongURL(longURL string) (string, error) {
	val, err := r.client.Get(r.ctx, "long:"+longURL).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("long URL not found")
	} else if err != nil {
		return "", fmt.Errorf("failed to get short URL by long URL from Redis: %w", err)
	}
	return val, nil
}

// ExtendTTL updates the expiration time of an existing key.
// It applies to both short and long URL mappings.
func (r *RedisClient) ExtendTTL(shortURL, longURL string, ttl time.Duration) error {
	// Extend TTL for shortURL -> longURL mapping
	err := r.client.Expire(r.ctx, "short:"+shortURL, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to extend TTL for short URL: %w", err)
	}

	// Extend TTL for longURL -> shortURL mapping
	err = r.client.Expire(r.ctx, "long:"+longURL, ttl).Err()
	if err != nil {
		fmt.Printf("Warning: failed to extend TTL for long URL mapping %s: %v\n", longURL, err)
	}
	return nil
}

// KeyExists checks if a key exists in Redis.
func (r *RedisClient) KeyExists(key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, "short:"+key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return count > 0, nil
}
