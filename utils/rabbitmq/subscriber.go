package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"prabogo/utils/log"
	"prabogo/utils/metrics"
)

type ExchangeKind string

const (
	KindFanOut  ExchangeKind = "fanout"
	KindTopic   ExchangeKind = "topic"
	KindDirect  ExchangeKind = "direct"
	KindHeaders ExchangeKind = "headers"
)

var (
	rabbitConn *amqp.Connection
)

func InitMessage() error {
	if rabbitConn != nil && !rabbitConn.IsClosed() {
		return nil
	}
	conn, err := dial()
	if err != nil {
		return err
	}
	rabbitConn = conn
	return nil
}

func CloseMessage() error {
	if rabbitConn == nil || rabbitConn.IsClosed() {
		rabbitConn = nil
		return nil
	}
	err := rabbitConn.Close()
	rabbitConn = nil
	return err
}

type SubscriberConfig struct {
	Exchange     string
	ExchangeKind ExchangeKind
	Queue        string
	RouteKey     string
	ExitCount    uint

	// Zero selects the package default; negative is invalid.
	PrefetchCount int
	PrefetchSize  int
	// Zero selects DefaultMaxBodyBytes, negative disables the check.
	MaxBodyBytes int64

	Callback func(ctx context.Context, msg []byte) bool
}

func (c *SubscriberConfig) Validate() error {
	if c.Exchange == "" {
		return errors.New("subscriber exchange empty")
	}
	if c.ExchangeKind == "" {
		return errors.New("subscriber exchange kind empty")
	}
	if c.Queue == "" {
		return errors.New("subscriber queue empty")
	}
	if c.Callback == nil {
		return errors.New("subscriber callback empty")
	}
	if err := validatePrefetch(c.PrefetchCount, c.PrefetchSize); err != nil {
		return fmt.Errorf("subscriber %w", err)
	}
	if c.MaxBodyBytes < -1 {
		return errors.New("subscriber max body bytes must be >= -1")
	}
	return nil
}

// prefetchCount returns the effective QoS count.
func (c *SubscriberConfig) prefetchCount() int {
	if c.PrefetchCount == 0 {
		return DefaultPrefetchCount
	}
	return c.PrefetchCount
}

// maxBodyLimit returns the effective body limit; 0 means unlimited.
func (c *SubscriberConfig) maxBodyLimit() int64 {
	switch {
	case c.MaxBodyBytes < 0:
		return 0
	case c.MaxBodyBytes == 0:
		return DefaultMaxBodyBytes
	default:
		return c.MaxBodyBytes
	}
}

func SubscriberWithConfig(cfg SubscriberConfig) error {
	if err := cfg.Validate(); err != nil {
		fmt.Printf("rabbitmq subscriber config error: %s\n", err.Error())
		return err
	}

	fmt.Printf("rabbitmq subscriber config: exchange=%q queue=%q routeKey=%q\n", cfg.Exchange, cfg.Queue, cfg.RouteKey)
	ctx := context.Background()

	if rabbitConn == nil || rabbitConn.IsClosed() {
		if err := InitMessage(); err != nil {
			return fmt.Errorf("failed to init rabbitmq connection: %w", err)
		}
	}

	ch, err := rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		errClose := ch.Close()
		if err == nil {
			err = errClose
		}
	}()

	err = ch.Qos(cfg.prefetchCount(), cfg.PrefetchSize, false)
	if err != nil {
		return err
	}

	err = ch.ExchangeDeclare(
		cfg.Exchange,
		string(cfg.ExchangeKind),
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		cfg.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		q.Name,
		cfg.RouteKey,
		cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	consumerKey := uuid.NewString()
	msgs, err := ch.Consume(
		q.Name,
		consumerKey,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	forever := make(chan struct{})
	go func() {
		maxBody := cfg.maxBodyLimit()
		for d := range msgs {
			if maxBody > 0 && int64(len(d.Body)) > maxBody {
				log.WithContext(ctx).Errorf(
					"dropping message on queue '%s' exceeding max body size: %d bytes > %d bytes (delivery tag %d)",
					cfg.Queue, len(d.Body), maxBody, d.DeliveryTag,
				)
				if nackErr := d.Nack(false, false); nackErr != nil {
					log.WithContext(ctx).Errorf("failed to nack oversized message: %s", nackErr)
				}
				continue
			}

			msgCtx := extractTraceContext(context.Background(), d.Headers)
			start := time.Now()
			ack := cfg.Callback(msgCtx, d.Body)
			metrics.RecordMessage(msgCtx, metrics.MessageRecord{
				Exchange:  cfg.Exchange,
				Queue:     cfg.Queue,
				Operation: "receive",
				Error:     !ack,
				Duration:  time.Since(start).Seconds(),
			})

			if ack {
				err = d.Ack(false)
				if err != nil {
					log.WithContext(ctx).Errorf("failed to ack message on queue '%s' (delivery tag %d): %s", cfg.Queue, d.DeliveryTag, err)
				}

			} else {
				err = d.Nack(false, true)
				if err != nil {
					log.WithContext(ctx).Errorf("failed to nack message on queue '%s' (delivery tag %d): %s", cfg.Queue, d.DeliveryTag, err)
				}
			}
		}
	}()
	log.WithContext(ctx).Infof("subscriber listen exchange: '%s', queue: '%s', topic: '%s', consumerKey: '%s'", cfg.Exchange, cfg.Queue, cfg.RouteKey, consumerKey)
	<-forever
	return nil
}

func extractTraceContext(parent context.Context, headers amqp.Table) context.Context {
	carrier := propagation.HeaderCarrier{}
	for k, v := range headers {
		if s, ok := v.(string); ok {
			carrier.Set(k, s)
		}
	}
	return otel.GetTextMapPropagator().Extract(parent, carrier)
}

func Subscriber(exchange string, exchangeKind ExchangeKind, queue, routeKey string, callback func(ctx context.Context, msg []byte) bool) error {
	return SubscriberWithConfig(SubscriberConfig{
		Exchange:     exchange,
		ExchangeKind: exchangeKind,
		Queue:        queue,
		RouteKey:     routeKey,
		ExitCount:    0,
		Callback:     callback,
	})
}
