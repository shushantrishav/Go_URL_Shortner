package shortener

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	// For a robust UUID generation, though not strictly needed for 6-digit
	"link-shortener/redis"
	"link-shortener/utils"
)

const (
	// ShortURLLength defines the length of the generated alphanumeric short URL.
	ShortURLLength = 6
	// URLExpiration defines the TTL for short URLs in Redis.
	URLExpiration = 7 * 24 * time.Hour // 7 days
)

// Shortener provides methods to shorten and retrieve URLs.
type Shortener struct {
	redisClient *redis.RedisClient
}

// NewShortener creates a new Shortener instance.
func NewShortener(rc *redis.RedisClient) *Shortener {
	return &Shortener{
		redisClient: rc,
	}
}

// GenerateRandomString generates a cryptographically secure random alphanumeric string of a given length.
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

// ShortenURL shortens a given long URL, optionally with a custom cshort.
// It handles repeated links by returning the existing short URL and extending its TTL.
func (s *Shortener) ShortenURL(longURL, customcshort string) (string, error) {
	// 1. Strict HTTPS check
	if !utils.IsValidHTTPS(longURL) {
		return "", fmt.Errorf("invalid or non-HTTPS URL: only HTTPS links are allowed and no scripts")
	}

	// 2. Check if the long URL has already been shortened
	existingShortURL, err := s.redisClient.GetShortURLByLongURL(longURL)
	if err == nil {
		// Long URL already exists, return the old entry and extend its life
		fmt.Printf("Long URL '%s' already shortened to '%s', extending TTL.\n", longURL, existingShortURL)
		err := s.redisClient.ExtendTTL(existingShortURL, longURL, URLExpiration)
		if err != nil {
			return "", fmt.Errorf("failed to extend TTL for existing URL: %w", err)
		}
		return existingShortURL, nil
	} else if err.Error() != "long URL not found" {
		// An actual error occurred other than "not found"
		return "", fmt.Errorf("failed to check for existing long URL: %w", err)
	}

	// 3. Determine the short cshort to use
	shortcshort := customcshort
	if shortcshort == "" {
		// Generate a new 6-digit alphanumeric string if no custom cshort is provided
		for {
			generatedcshort, err := GenerateRandomString(ShortURLLength)
			if err != nil {
				return "", fmt.Errorf("failed to generate random string: %w", err)
			}
			exists, err := s.redisClient.KeyExists(generatedcshort)
			if err != nil {
				return "", fmt.Errorf("failed to check cshort existence: %w", err)
			}
			if !exists {
				shortcshort = generatedcshort
				break // Found a unique cshort
			}
			// If exists, loop again to generate a new one
		}
	} else {
		// If custom cshort is provided, check if it's already taken
		exists, err := s.redisClient.KeyExists(customcshort)
		if err != nil {
			return "", fmt.Errorf("failed to check custom cshort existence: %w", err)
		}
		if exists {
			return "", fmt.Errorf("custom cshort '%s' is already in use", customcshort)
		}
	}

	// 4. Store the mapping in Redis with TTL
	err = s.redisClient.SetURL(shortcshort, longURL, URLExpiration)
	if err != nil {
		return "", fmt.Errorf("failed to save short URL to Redis: %w", err)
	}

	return shortcshort, nil
}

// GetLongURL retrieves the original long URL for a given short cshort.
func (s *Shortener) GetLongURL(shortcshort string) (string, error) {
	return s.redisClient.GetLongURL(shortcshort)
}
