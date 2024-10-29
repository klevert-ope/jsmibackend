package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

type RedisConfig struct {
	URL          string
	PoolSize     int
	DialTimeout  time.Duration
	MinIdleConns int
	ReadTimeout  time.Duration
	MaxRetries   int
}

// LoadRedisConfig loads the Redis configuration from environment variables.
func LoadRedisConfig() (RedisConfig, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return RedisConfig{}, errors.New("REDIS_URL environment variable is not set")
	}

	return RedisConfig{
		URL:          redisURL,
		PoolSize:     10,
		DialTimeout:  30 * time.Second,
		MinIdleConns: 5,
		ReadTimeout:  30 * time.Second,
		MaxRetries:   3,
	}, nil
}

// NewRedisClient creates a new Redis client based on the provided configuration.
func NewRedisClient(config RedisConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}
	opt.DialTimeout = config.DialTimeout

	client := redis.NewClient(opt)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis server: %v", err)
	}

	client.Options().PoolSize = config.PoolSize
	client.Options().MinIdleConns = config.MinIdleConns
	client.Options().ReadTimeout = config.ReadTimeout
	client.Options().MaxRetries = config.MaxRetries

	return client, nil
}

// InitRedis initializes the Redis client and logs the connection status.
func InitRedis() error {
	config, err := LoadRedisConfig()
	if err != nil {
		return fmt.Errorf("failed to load Redis configuration: %v", err)
	}

	RedisClient, err = NewRedisClient(config)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis client: %v", err)
	}

	log.Println("Redis connection initialized successfully.")
	return nil
}
