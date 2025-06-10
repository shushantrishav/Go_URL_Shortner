package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"link-shortener/ratelimiter"
	"link-shortener/shortener"
)

// ShortenRequest represents the JSON structure for a shorten request.
type ShortenRequest struct {
	LongURL    string `json:"long_url"`
	Customcshort string `json:"custom_cshort,omitempty"`
}

// ShortenResponse represents the JSON structure for a shorten response.
type ShortenResponse struct {
	ShortURL      string `json:"short_url"`
	LongURL       string `json:"long_url,omitempty"`      // Added LongURL
	LimitRemaining int64  `json:"limit_remaining,omitempty"` // Added LimitRemaining
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

// LinkShortenerHandler handles the URL shortening requests.
type LinkShortenerHandler struct {
	shortener   *shortener.Shortener
	rateLimiter *ratelimiter.RateLimiter
}

// NewLinkShortenerHandler creates a new LinkShortenerHandler.
func NewLinkShortenerHandler(s *shortener.Shortener, rl *ratelimiter.RateLimiter) *LinkShortenerHandler {
	return &LinkShortenerHandler{
		shortener:   s,
		rateLimiter: rl,
	}
}

// Shorten handles the POST request to shorten a URL.
func (h *LinkShortenerHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	// Set Content-Type header for JSON response
	w.Header().Set("Content-Type", "application/json")

	// Apply rate limiting based on client's IP address
	clientIP := r.RemoteAddr
	allowed, currentCount, err := h.rateLimiter.Allow(clientIP) // Capture currentCount
	if err != nil {
		http.Error(w, `{"error": "Internal server error during rate limiting"}`, http.StatusInternalServerError)
		fmt.Printf("Rate limiter error for IP %s: %v\n", clientIP, err)
		return
	}

	remaining := ratelimiter.MaxRequests - currentCount
	if remaining < 0 {
		remaining = 0 // Ensure remaining doesn't go negative if max requests is exceeded
	}

	if !allowed {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ShortenResponse{
			Error:          fmt.Sprintf("Rate limit exceeded: %d URLs per %s", ratelimiter.MaxRequests, ratelimiter.RateLimitDuration),
			LimitRemaining: remaining, // Include remaining limit in error response
		})
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ShortenResponse{Error: "Invalid request payload"})
		return
	}

	if req.LongURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ShortenResponse{Error: "long_url is required"})
		return
	}

	shortcshort, err := h.shortener.ShortenURL(req.LongURL, req.Customcshort)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid or non-HTTPS URL") || strings.Contains(err.Error(), "custom cshort") {
			statusCode = http.StatusBadRequest
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(ShortenResponse{
			Error:          err.Error(),
			LimitRemaining: remaining, // Include remaining limit in error response
		})
		return
	}

	fullShortURL := fmt.Sprintf("/s/%s", shortcshort)

	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL:       fullShortURL,
		LongURL:        req.LongURL,       // Return the original long URL
		LimitRemaining: remaining,         // Return the calculated remaining limit
		Message:        "URL shortened successfully",
	})
}

// Redirect handles the GET request to redirect from a short URL to the original long URL.
func (h *LinkShortenerHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	// Extract the short cshort from the URL path.
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[2] == "" {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}
	shortcshort := parts[2]

	longURL, err := h.shortener.GetLongURL(shortcshort)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Short URL not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			fmt.Printf("Error retrieving long URL for cshort '%s': %v\n", shortcshort, err)
		}
		return
	}

	http.Redirect(w, r, longURL, http.StatusMovedPermanently)
}
