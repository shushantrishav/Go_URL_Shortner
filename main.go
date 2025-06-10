package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"link-shortener/config"
	"link-shortener/handler"
	"link-shortener/ratelimiter"
	redisClient "link-shortener/redis"
	"link-shortener/shortener"
)

func main() {
	// Load .env file for local development
	if os.Getenv("GO_ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Note: Error loading .env file. Assuming environment variables are set externally or defaults will be used:", err)
		}
	}

	// 1. Load Configuration
	cfg := config.LoadConfig()

	// Configure TLS for Redis connection. Upstash typically requires this.
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// InsecureSkipVerify: true, // For development only if certificate issues arise with Redis (use with caution!)
	}

	// 2. Initialize Redis Client
	redisNativeClient := redis.NewClient(&redis.Options{
		Addr:      cfg.RedisAddr,
		Password:  cfg.RedisPassword,
		DB:        cfg.RedisDB,
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

	// --- Define Specific API Routes FIRST ---

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	}).Methods("GET")

	// Shorten URL endpoint (POST /shorten)
	r.HandleFunc("/shorten", linkHandler.Shorten).Methods("POST")

	// Redirect endpoint (GET /s/{short_slug})
	// This route MUST be defined explicitly and before any catch-all static file server
	r.HandleFunc("/s/{short_slug}", linkHandler.Redirect).Methods("GET")

	// --- Handle the Root Path (API Home Page) ---
	// Serve index.html directly for the root path
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	}).Methods("GET")

	// Configure CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler(r)

	// 6. Start the HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.HTTPPort
	}
	serverAddr := fmt.Sprintf(":%s", port)

	log.Printf("Server starting on %s...", serverAddr)
	log.Printf("Access API Home Page at: http://localhost:%s/", port)
	log.Printf("Access shorten endpoint at: http://localhost:%s/shorten (POST)", port)
	log.Printf("Access redirect endpoint at: http://localhost:%s/s/{your_slug} (GET)", port)
	log.Printf("Access health check at: http://localhost:%s/health (GET)", port)

	// Use the CORS-wrapped handler here
	log.Fatal(http.ListenAndServe(serverAddr, c))
}
