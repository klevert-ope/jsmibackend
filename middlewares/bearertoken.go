package middlewares

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// ValidateBearerToken validates the Bearer token in the Authorization header.
func ValidateBearerToken(expectedBearerToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the Bearer token from the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
				return
			}

			// Check if the Authorization header has the Bearer scheme
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Extract the token from the Authorization header
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Convert both tokens to lowercase for case-insensitive comparison
			expectedTokenLower := strings.ToLower(expectedBearerToken)
			tokenLower := strings.ToLower(token)

			// Constant-time comparison to mitigate timing attacks
			if !secureCompare(tokenLower, expectedTokenLower) {
				http.Error(w, "Invalid Bearer Token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// secureCompare performs a constant-time comparison of two strings.
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// LoggingMiddleware logs information about incoming requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
