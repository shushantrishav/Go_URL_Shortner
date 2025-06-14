package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"crypto/tls"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors" // Import the CORS library

	"link-shortener/config"
	"link-shortener/handler"
	redisClient "link-shortener/redis"
	"link-shortener/ratelimiter"
	"link-shortener/shortener"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// Configure TLS for Redis connection. Upstash typically requires this.
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// InsecureSkipVerify: true, // For development only if certificate issues arise with Redis
	}

	// 2. Initialize Redis Client
	redisNativeClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		TLSConfig: tlsConfig,
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

	// Redirect endpoint (GET /s/{short_slug})
	r.HandleFunc("/s/{short_slug}", linkHandler.Redirect).Methods("GET")

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")


	// Configure CORS middleware
	// For production, replace cors.AllowAll() with more specific origins and methods.
	// Example: cors.New(cors.Options{
	// 	AllowedOrigins: []string{"http://localhost:5500", "https://your-frontend-app.com"}, // Replace with your actual frontend origins
	// 	AllowedMethods: []string{"GET", "POST", "OPTIONS"}, // Add all methods your API uses
	// 	AllowedHeaders: []string{"Content-Type"}, // Add all headers your API expects
	// 	AllowCredentials: true, // Set to true if you are using cookies/authentication
	// }).Handler(r)
	c := cors.AllowAll().Handler(r) // Allows all origins, methods, and headers (simplest for dev)


	// 6. Start the HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.HTTPPort // Fallback to HTTP_PORT from config for local dev
	}
	serverAddr := fmt.Sprintf(":%s", port)

	log.Printf("Server starting on %s...", serverAddr)
	log.Printf("Access shorten endpoint at: http://localhost:%s/shorten (POST)", port)
	log.Printf("Access redirect endpoint at: http://localhost:%s/s/{your_slug} (GET)", port)

	// Use the CORS-wrapped handler here
	log.Fatal(http.ListenAndServe(serverAddr, c)) // Pass 'c' (the CORS handler)
}
