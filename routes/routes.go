package routes

import (
	"jsmi-api/controllers"
	"jsmi-api/db"
	"jsmi-api/middlewares"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gorilla/mux"
)

// Config interface represents the configuration needed for setting up routes.
type Config interface {
	GetBearerToken() string
}

// SetupRoutes sets up the application routes and middlewares.
func SetupRoutes(config Config) http.Handler {
	router := mux.NewRouter()
	authHandler := &controllers.AuthHandler{
		Config: config.(*db.Config), // Assert the type to *db.Config
	}

	// Apply global middlewares
	router.Use(middlewares.CorsMiddleware(&middlewares.CorsConfig{
		AllowedOrigins:   []string{"http://0.0.0.0:3000", "http://localhost:8000", "https://www.jehovahshammahministriesinternational.org", "https://jsmi.koyeb.app"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	router.Use(middlewares.LoggingMiddleware)

	// Initialize rate limiter and apply to all routes
	rateLimiter := middlewares.NewRateLimiter(30, time.Minute, 2*time.Minute)
	router.Use(rateLimiter.Limit)

	// Set up protected routes (apply Bearer token middleware here)
	protectedRouter := router.PathPrefix("/").Subrouter()
	protectedRouter.Use(middlewares.ValidateBearerToken(config.GetBearerToken()))

	// Set up routes that require authentication
	controllers.SetupRootRoute(protectedRouter)
	controllers.SetupPostRoutes(protectedRouter)
	controllers.SetupLiveRoutes(protectedRouter)
	authHandler.SetupUserRoutes(protectedRouter)

	// Register pprof routes to enable profiling
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	return router
}
