package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
	"time"
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

func init() {
	config, err := LoadRedisConfig()
	if err != nil {
		log.Fatalf("Failed to load Redis configuration: %v", err)
	}

	RedisClient, err = NewRedisClient(config)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	log.Println("Redis connection initialized successfully.")
}

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
