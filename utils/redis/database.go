package redis

import (
	"context"
	"os"

	redis "github.com/redis/go-redis/v9"
)

var dbClient *redis.Client

func InitDatabase() {
	addr := os.Getenv("CACHE_HOST")
	port := os.Getenv("CACHE_PORT")
	pass := os.Getenv("CACHE_PASSWORD")
	if port == "" {
		port = "6379"
	}
	dbClient = redis.NewClient(&redis.Options{
		Addr:     addr + ":" + port,
		Password: pass,
	})
}

func Set(ctx context.Context, key string, value interface{}) error {
	return dbClient.Set(ctx, key, value, 24*60*60*1e9).Err() // 1 day in nanoseconds
}

func Get(ctx context.Context, key string) (string, error) {
	return dbClient.Get(ctx, key).Result()
}

func Del(ctx context.Context, key string) error {
	return dbClient.Del(ctx, key).Err()
}
