package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/novapanel/novapanel/internal/config"
)

func NewRedisClient(cfg *config.Config) *redis.Client {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("✓ Connected to Redis")
	return client
}
