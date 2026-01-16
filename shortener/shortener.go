package shortener

import (
    "crypto/rand"
    "fmt"
    "math/big"
    "time"

    rdb "github.com/go-redis/redis/v8"
    "link-shortener/redis"
    "link-shortener/utils"
)

const (
    ShortURLLength = 6
    URLExpiration = 30 * 24 * time.Hour // 30 days ttl
    LongURLPrefix  = "long:"
    ShortURLPrefix = "short:"
)

type Shortener struct {
    redisClient *redis.RedisClient
}

func NewShortener(rc *redis.RedisClient) *Shortener {
    return &Shortener{redisClient: rc}
}

func GenerateRandomString(length int) (string, error) {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := 0; i < length; i++ {
        num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        b[i] = charset[num.Int64()]
    }
    return string(b), nil
}

const atomicShortenScript = `
local url_key = KEYS[1]           
local slug_key = KEYS[2]          
local long_url = ARGV[1]
local slug = ARGV[2]
local ttl = ARGV[3]
local is_custom = ARGV[4] == "1"  

local existing_slug = redis.call('GET', url_key)
if existing_slug then
    redis.call('EXPIRE', url_key, ttl, 'GT')
    redis.call('EXPIRE', 'short:' .. existing_slug, ttl, 'GT')
    return { 'EXISTS', existing_slug }
end

local set_result = redis.call('SET', slug_key, long_url, 'NX', 'EX', ttl)
if set_result then
    redis.call('SET', url_key, slug, 'EX', ttl)
    return { 'CREATED', slug }
end

if is_custom then
    return { 'SLUG_TAKEN', '' }
else
    return { 'COLLISION', '' }
end
`

func (s *Shortener) ShortenURL(longURL, customSlug string) (string, error) {
    if !utils.IsValidHTTPS(longURL) {
        return "", fmt.Errorf("invalid or non-HTTPS URL: only HTTPS links are allowed and no scripts")
    }

    urlKey := LongURLPrefix + longURL

    for {
        slug := customSlug
        isCustom := customSlug != ""

        if !isCustom {
            var err error
            slug, err = GenerateRandomString(ShortURLLength)
            if err != nil {
                return "", fmt.Errorf("failed to generate random string: %w", err)
            }
        }

        slugKey := ShortURLPrefix + slug

        result, err := s.redisClient.Client().Eval(s.redisClient.Context(), atomicShortenScript, []string{urlKey, slugKey}, longURL, slug, URLExpiration, isCustom).Result()
        if err != nil {
            return "", fmt.Errorf("Redis script execution failed: %w", err)
        }

        resArray := result.([]interface{})
        if len(resArray) != 2 {
            return "", fmt.Errorf("invalid script response length: %d", len(resArray))
        }
        
        status := resArray[0].(string)
        slugResult := resArray[1].(string)

        switch status {
        case "EXISTS":
            fmt.Printf("Long URL '%s' already shortened to '%s', TTL refreshed.\n", longURL, slugResult)
            return slugResult, nil
        case "CREATED":
            fmt.Printf("Created new mapping: '%s' -> '%s'\n", slugResult, longURL)
            return slugResult, nil
        case "SLUG_TAKEN":
            return "", fmt.Errorf("custom slug '%s' is already in use", slug)
        case "COLLISION":
            if isCustom {
                return "", fmt.Errorf("custom slug '%s' is already in use", slug)
            }
            continue
        default:
            return "", fmt.Errorf("unexpected script result: %v", status)
        }
    }
}

func (s *Shortener) GetLongURL(shortSlug string) (string, error) {
    slugKey := ShortURLPrefix + shortSlug
    longURL, err := s.redisClient.Client().Get(s.redisClient.Context(), slugKey).Result()
    if err == rdb.Nil {
        return "", fmt.Errorf("short URL not found")
    } else if err != nil {
        return "", fmt.Errorf("failed to get long URL from Redis: %w", err)
    }
    
    s.redisClient.Client().Expire(s.redisClient.Context(), slugKey, URLExpiration).Err()
    return longURL, nil
}
