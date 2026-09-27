package database

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"task-management-backend/config"
)

// NewRedis creates a go-redis client and verifies it with a PING.
func NewRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	log.Println("connected to redis")
	return client
}
