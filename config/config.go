package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config struct holds the application's configuration settings.
type Config struct {
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	HTTPPort        string
	HTTPSPort       string
	TLSCertPath     string
	TLSKeyPath      string
	AllowedOrigins  []string
}

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper to get environment variable as an int with a default value
func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("Warning: Could not parse %s as int, using default %d. Error: %v", key, defaultValue, err)
		return defaultValue
	}
	return intValue
}

// Helper to get environment variable as a slice of strings
func getEnvAsSlice(key string, defaultValue string) []string {
	value := getEnv(key, defaultValue)
	if value == "" {
		return []string{}
	}
	// Split the string by comma and trim whitespace from each part
	parts := strings.Split(value, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	redisAddr := getEnv("REDIS_ADDR", "")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR environment variable not set")
	}

	redisPassword := getEnv("REDIS_PASSWORD", "")

	redisDB := getEnvAsInt("REDIS_DB", 0)

	httpPort := getEnv("HTTP_PORT", "8080")
	httpsPort := getEnv("HTTPS_PORT", "8443")
	tlsCertPath := getEnv("TLS_CERT_PATH", "")
	tlsKeyPath := getEnv("TLS_KEY_PATH", "")

	allowedOrigins := getEnvAsSlice(
		"AllowedOrigins", // Matches your .env key
		"", // No default provided here, as you expect it from .env or for it to be empty
	)

	return &Config{
		RedisAddr:      redisAddr,
		RedisPassword:  redisPassword,
		RedisDB:        redisDB,
		HTTPPort:       httpPort,
		HTTPSPort:      httpsPort,
		TLSCertPath:    tlsCertPath,
		TLSKeyPath:     tlsKeyPath,
		AllowedOrigins: allowedOrigins,
	}
}