package db

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient создаёт подключение к Redis
func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("✅ Подключение к Redis установлено")
	return client, nil
}