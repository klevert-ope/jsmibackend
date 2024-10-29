package middlewares

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

// LoadBearerTokenConfig retrieves the bearer token from the environment variable.
func LoadBearerTokenConfig() (string, error) {
	bearerToken := os.Getenv("BEARER_TOKEN")
	if bearerToken == "" {
		return "", errors.New("bearer token environment variable (BEARER_TOKEN) is not set")
	}
	return bearerToken, nil
}

// ValidateBearerToken validates the Bearer token in the Authorization header.
func ValidateBearerToken() func(http.Handler) http.Handler {
	// Load the Bearer token when the middleware is initialized
	expectedBearerToken, err := LoadBearerTokenConfig()
	if err != nil {
		log.Fatalf("Failed to load Bearer token: %v", err)
	}

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
	for i := range a {
		result |= a[i] ^ b[i]
	}

	return result == 0
}
