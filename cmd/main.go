package main

import (
	"context"
	"errors"
	"jsmi-api/db"
	"jsmi-api/routes"
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
	config, err := db.LoadEnvConfig()
	if err != nil {
		log.Fatalf("failed to load ENV configuration: %v", err)
	}

	// Migrate the database
	migrateCfg := db.MigrateConfig{
		DBURL: config.DBURL,
	}
	if err := db.Migrate(migrateCfg); err != nil {
		log.Fatalf("error migrating database: %v", err)
	}

	// Set up routes and middlewares
	handler := routes.SetupRoutes(config)

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
			log.Fatalf("listenAndServe(): %v", err)
		}
	}()
	log.Println("server started on :8000")

	// Wait for interrupt signal to gracefully shut down the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Create a context with a timeout for shutdown
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}

	wg.Wait() // Wait for all goroutines to finish before exiting
	log.Println("server exited gracefully")
}
