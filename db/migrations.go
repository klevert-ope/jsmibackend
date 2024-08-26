package db

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
)

// MigrateConfig defines the configuration needed for database migrations
type MigrateConfig struct {
	DBURL string
}

// Migrate runs the database migrations using the provided configuration
func Migrate(cfg MigrateConfig) error {
	// Create a context with a timeout for database initialization
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the database connection with the context
	if err := InitDB(ctx, cfg.DBURL); err != nil {
		return errors.New("failed to initialize database: " + err.Error())
	}

	// Get the absolute path to the migrations directory
	migrationsDir, err := filepath.Abs("db/migrations")
	if err != nil {
		return errors.New("failed to get absolute path to migrations directory: " + err.Error())
	}

	// Run database migrations
	if err := goose.SetDialect("postgres"); err != nil {
		return errors.New("failed to set dialect: " + err.Error())
	}

	if err := goose.Up(DB, migrationsDir); err != nil {
		return errors.New("failed to run migrations: " + err.Error())
	}

	log.Println("database migration check complete. All migrations are up to date")
	return nil
}
