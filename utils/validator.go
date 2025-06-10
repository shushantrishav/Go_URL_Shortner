package utils

import (
	"net/url"
	"strings"
)

// IsValidHTTPS checks if a given string is a valid URL and uses the HTTPS scheme.
// It also performs a basic check to prevent obvious script injection attempts.
func IsValidHTTPS(rawURL string) bool {
	// Trim whitespace to avoid issues with extra spaces
	rawURL = strings.TrimSpace(rawURL)

	// Basic check for common script injection keywords (not exhaustive, but adds a layer)
	if strings.ContainsAny(rawURL, "<>") ||
		strings.Contains(strings.ToLower(rawURL), "javascript:") ||
		strings.Contains(strings.ToLower(rawURL), "data:") ||
		strings.Contains(strings.ToLower(rawURL), "php:") {
		return false
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false // Not a valid URI
	}

	// Check if the scheme is "https" and not empty
	return parsedURL.Scheme == "https"
}
