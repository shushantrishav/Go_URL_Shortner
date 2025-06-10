package main

import (
	"crypto/tls" // Import the tls package
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8" // Using V8 of go-redis client
	"github.com/gorilla/mux"       // A powerful URL router for Go

	"link-shortener/config"
	"link-shortener/handler"
	"link-shortener/ratelimiter"
	redisClient "link-shortener/redis" // Alias to avoid conflict with go-redis
	"link-shortener/shortener"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// Configure TLS for Redis connection. Upstash typically requires this.
	// We keep this as it's for the Redis connection, not the HTTP server.
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// InsecureSkipVerify: true, // IMPORTANT: Set to true only for development if you encounter certificate issues.
		// For production, remove this line and ensure proper CA certs are installed if needed.
	}

	// 2. Initialize Redis Client
	redisNativeClient := redis.NewClient(&redis.Options{
		Addr:      cfg.RedisAddr,
		Password:  cfg.RedisPassword,
		DB:        cfg.RedisDB,
		TLSConfig: tlsConfig, // Add TLS configuration here
	})

	_, err := redisNativeClient.Ping(redisNativeClient.Context()).Result()
	if err != nil {
		log.Fatalf("Could not connect to native Redis client: %v", err)
	}
	log.Println("Native Redis client connected successfully.")

	redisWrapperClient, err := redisClient.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client wrapper: %v", err)
	}

	// 3. Initialize Shortener and Rate Limiter services
	linkShortener := shortener.NewShortener(redisWrapperClient)
	rateLimiter := ratelimiter.NewRateLimiter(redisNativeClient)

	// 4. Initialize HTTP Handlers
	linkHandler := handler.NewLinkShortenerHandler(linkShortener, rateLimiter)

	// 5. Setup Router using Gorilla Mux
	r := mux.NewRouter()

	// Shorten URL endpoint (POST /shorten)
	r.HandleFunc("/shorten", linkHandler.Shorten).Methods("POST")

	// Redirect endpoint (GET /s/{short_cshort})
	r.HandleFunc("/s/{short_cshort}", linkHandler.Redirect).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	// 6. Start the HTTP Server
	// Render provides a 'PORT' environment variable. We should listen on that.
	// Default to HTTP_PORT for local development if PORT is not set.
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.HTTPPort // Fallback to HTTP_PORT from config for local dev
	}
	serverAddr := fmt.Sprintf(":%s", port)

	log.Printf("Server starting on %s...", serverAddr)
	log.Printf("Access shorten endpoint at: http://localhost:%s/shorten (POST)", port)
	log.Printf("Access redirect endpoint at: http://localhost:%s/s/{your_cshort} (GET)", port)

	// We use ListenAndServe directly. Render will handle HTTPS termination.
	log.Fatal(http.ListenAndServe(serverAddr, r))
}
