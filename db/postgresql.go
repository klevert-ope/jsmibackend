package db

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
)

var DB *sql.DB

// InitDB initializes the database connection and sets up Goose for migrations
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

// Config holds the application configuration.
type Config struct {
	DBURL       string
	BearerToken string
}

// GetBearerToken retrieves the bearer token from the configuration.
func (c *Config) GetBearerToken() string {
	return c.BearerToken
}

func LoadEnvConfig() (*Config, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errors.New("database URL (DB_URL) environment variable is not set")
	}

	bearerToken := os.Getenv("BEARER_TOKEN")
	if bearerToken == "" {
		return nil, errors.New("bearer token environment variable (BEARER_TOKEN) is not set")
	}

	return &Config{
		DBURL:       dbURL,
		BearerToken: bearerToken,
	}, nil
}
