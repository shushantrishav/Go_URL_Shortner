package config

import (
	"log"
	"os"
	"strconv"
)

// Config struct holds the application's configuration settings.
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	HTTPPort      string
	HTTPSPort     string // New field for HTTPS port
	TLSCertPath   string // New field for TLS certificate path
	TLSKeyPath    string // New field for TLS key path
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() *Config {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR environment variable not set")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	// Password can be empty if Redis is not secured, but generally recommended to set one.

	redisDBStr := os.Getenv("REDIS_DB")
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Printf("REDIS_DB environment variable not set or invalid, defaulting to 0: %v", err)
		redisDB = 0 // Default to DB 0 if not set or invalid
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		log.Println("HTTP_PORT environment variable not set, defaulting to 8080")
		httpPort = "8080" // Default to port 8080
	}

	httpsPort := os.Getenv("HTTPS_PORT") // Load HTTPS port
	if httpsPort == "" {
		log.Println("HTTPS_PORT environment variable not set, defaulting to 8443")
		httpsPort = "8443" // Default to port 8443
	}

	tlsCertPath := os.Getenv("TLS_CERT_PATH")
	tlsKeyPath := os.Getenv("TLS_KEY_PATH")

	return &Config{
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
		HTTPPort:      httpPort,
		HTTPSPort:     httpsPort,   // Assign HTTPS port
		TLSCertPath:   tlsCertPath, // Assign TLS cert path
		TLSKeyPath:    tlsKeyPath,  // Assign TLS key path
	}
}
