package db

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

var DB *sql.DB

type Config struct {
	DBURL string
}

// InitDB initializes the database connection and sets up Goose for migrations.
func InitDB(ctx context.Context, dataSourceName string) error {
	var err error
	DB, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return errors.New("failed to open database connection: " + err.Error())
	}

	// Check database connection
	if err = DB.PingContext(ctx); err != nil {
		return errors.New("failed to ping database: " + err.Error())
	}

	// Set up Goose to use the database connection
	if err := goose.SetDialect("postgres"); err != nil {
		return errors.New("failed to set Goose dialect: " + err.Error())
	}

	// Configure database connection pool settings
	DB.SetMaxOpenConns(20)
	DB.SetMaxIdleConns(10)

	log.Println("Database connection initialized successfully.")
	return nil
}

// LoadDBConfig retrieves the database URL from environment variables.
func LoadDBConfig() (*Config, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errors.New("database URL (DB_URL) environment variable is not set")
	}

	return &Config{
		DBURL: dbURL,
	}, nil
}
