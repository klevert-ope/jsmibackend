package main

import (
	"context"
	"errors"
	"jsmi-api/db"
	"jsmi-api/middlewares"
	"jsmi-api/routes"
	"jsmi-api/utils"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Load configuration
	config, err := db.LoadDBConfig()
	if err != nil {
		log.Fatalf("Error loading database config: %v", err)
	}

	envCheck()

	// Initialize Redis
	if err := db.InitRedis(); err != nil {
		log.Fatalf("Error initializing Redis: %v", err)
	}

	// Migrate the database
	migrateCfg := db.MigrateConfig{
		DBURL: config.DBURL,
	}

	if err := db.Migrate(migrateCfg); err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}

	// Set up routes and middlewares
	handler := routes.SetupRoutes(config)

	// Wrap the handler with the bearer token middleware
	handler = middlewares.ValidateBearerToken()(handler)

	srv := &http.Server{
		Addr:           ":8000",
		Handler:        handler,
		ReadTimeout:    100 * time.Second,
		WriteTimeout:   100 * time.Second,
		MaxHeaderBytes: 7500,
		IdleTimeout:    120 * time.Second,
	}

	// Use a wait group to manage graceful shutdown
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()
	log.Println("Server started on :8000")

	// Wait for interrupt signal to gracefully shut down the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Create a context with a timeout for shutdown
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}

	wg.Wait() // Wait for all goroutines to finish before exiting
	log.Println("Server exited gracefully")
}

func envCheck() {
	// Check bearer token environment variable
	if _, err := middlewares.LoadBearerTokenConfig(); err != nil {
		log.Fatalf("Error loading bearer token: %v", err)
	} else {
		log.Println("Bearer token environment variable is set.")
	}

	// Check Redis configuration
	if _, err := db.LoadRedisConfig(); err != nil {
		log.Fatalf("Error loading Redis config: %v", err)
	} else {
		log.Println("Redis configuration environment variable is set.")
	}

	// Check PASETO secret environment variable
	if _, err := utils.GetPasetoSecret(); err != nil {
		log.Fatalf("Error retrieving PASETO secret: %v", err)
	} else {
		log.Println("PASETO secret environment variable is set.")
	}
}
