package middlewares

import (
	"net/http"
	"strings"
)

// CorsConfig holds CORS configuration settings.
type CorsConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// CorsMiddleware creates a CORS middlewares based on the provided configuration.
func CorsMiddleware(config *CorsConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if contains(config.AllowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", commaSeparated(config.AllowedMethods))
			w.Header().Set("Access-Control-Allow-Headers", commaSeparated(config.AllowedHeaders))
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "deny")
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}

func commaSeparated(arr []string) string {
	return strings.Join(arr, ", ")
}
