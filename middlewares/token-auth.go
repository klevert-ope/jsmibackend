package middlewares

import (
	"jsmi-api/utils"
	"net/http"
	"os"
)

// TokenAuthMiddleware is a middleware function that checks for a valid PASETO token
func TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil || cookie == nil || cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		pasetoSecret := os.Getenv("PASETO_SECRET")
		if pasetoSecret == "" {
			http.Error(w, "Server configuration error", http.StatusInternalServerError)
			return
		}

		_, err = utils.ValidatePASETO(cookie.Value, pasetoSecret)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
