package redis

import (
	"context"
	"os"

	redis "github.com/redis/go-redis/v9"
)

var pubsubClient *redis.Client

func InitPubsub() {
	addr := os.Getenv("MESSAGE_HOST")
	port := os.Getenv("MESSAGE_PORT")
	pass := os.Getenv("MESSAGE_PASSWORD")
	if port == "" {
		port = "6379"
	}
	pubsubClient = redis.NewClient(&redis.Options{
		Addr:     addr + ":" + port,
		Password: pass,
	})
}

func Publish(ctx context.Context, channel string, message string) error {
	return pubsubClient.Publish(ctx, channel, message).Err()
}

func Subscribe(ctx context.Context, channel string, handler func(string)) error {
	pubsub := pubsubClient.Subscribe(ctx, channel)
	ch := pubsub.Channel()
	for msg := range ch {
		handler(msg.Payload)
	}
	return nil
}
