package google

import (
	"context"
	"os"
	"strconv"
	"sync"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

var (
	pubsubClient *pubsub.Client
	pubsubOnce   sync.Once
)

func InitMessage(ctx context.Context) error {
	var err error
	pubsubOnce.Do(func() {
		projectID := os.Getenv("GOOGLE_PROJECT_ID")
		credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if projectID == "" {
			err = ErrMissingProjectID
			return
		}
		if credsFile == "" {
			err = ErrMissingCredentials
			return
		}
		pubsubClient, err = pubsub.NewClient(ctx, projectID, option.WithCredentialsFile(credsFile))
	})
	return err
}

// GetPubSubClient returns the initialized Pub/Sub client.
func GetPubSubClient() *pubsub.Client {
	return pubsubClient
}

// Publish publishes a message to a topic.
func Publish(ctx context.Context, topicName string, data []byte, attrs map[string]string) (string, error) {
	if pubsubClient == nil {
		return "", ErrClientNotInitialized
	}
	topic := pubsubClient.Topic(topicName)
	result := topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})
	return result.Get(ctx)
}

// Subscribe subscribes to a subscription and handles messages with the given callback.
// It supports setting max outstanding messages via env GOOGLE_PUBSUB_SUB_MAX_OUTSTANDING_MESSAGES (default: 5)
func Subscribe(ctx context.Context, subscriptionName string, handler func(ctx context.Context, msg *pubsub.Message)) error {
	if pubsubClient == nil {
		return ErrClientNotInitialized
	}
	sub := pubsubClient.Subscription(subscriptionName)

	// Set max outstanding messages from env
	maxOutstanding := 5
	if envVal := os.Getenv("GOOGLE_PUBSUB_SUB_MAX_OUTSTANDING_MESSAGES"); envVal != "" {
		if v, err := strconv.Atoi(envVal); err == nil && v > 0 {
			maxOutstanding = v
		}
	}
	sub.ReceiveSettings.MaxOutstandingMessages = maxOutstanding

	return sub.Receive(ctx, handler)
}

// Error variables
var (
	ErrMissingProjectID     = errorNew("GOOGLE_PROJECT_ID env is required")
	ErrMissingCredentials   = errorNew("GOOGLE_APPLICATION_CREDENTIALS env is required")
	ErrClientNotInitialized = errorNew("Google Pub/Sub client not initialized")
)

func errorNew(msg string) error {
	return &pubsubError{msg: msg}
}

type pubsubError struct {
	msg string
}

func (e *pubsubError) Error() string {
	return e.msg
}
