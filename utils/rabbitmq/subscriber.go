package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"prabogo/utils/log"
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

func getUrl() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		os.Getenv("MESSAGE_USER"),
		os.Getenv("MESSAGE_PASSWORD"),
		os.Getenv("MESSAGE_HOST"),
		os.Getenv("MESSAGE_PORT"),
		os.Getenv("MESSAGE_VHOST"),
	)
}

func InitMessage() error {
	if rabbitConn != nil && !rabbitConn.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(getUrl())
	if err != nil {
		return err
	}
	rabbitConn = conn
	return nil
}

type SubscriberConfig struct {
	Exchange     string
	ExchangeKind ExchangeKind
	Queue        string
	RouteKey     string
	ExitCount    uint
	Callback     func(msg []byte) bool
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
	return nil
}

func SubscriberWithConfig(cfg SubscriberConfig) error {
	if err := cfg.Validate(); err != nil {
		fmt.Printf("rabbitmq subscriber config error: %s\n", err.Error())
		return err
	}

	fmt.Printf("rabbitmq subscriber config: %+v\n", cfg)
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
		for d := range msgs {
			ack := cfg.Callback(d.Body)
			if ack {
				err = d.Ack(false)
				if err != nil {
					log.WithContext(context.Background()).Errorf("failed to ack message with body %s: %s", string(d.Body), err)
				}

			} else {
				err = d.Nack(false, true)
				if err != nil {
					log.WithContext(context.Background()).Errorf("failed to nack message with body %s: %s", string(d.Body), err)
				}
			}
		}
	}()
	log.WithContext(ctx).Infof("subscriber listen exchange: '%s', queue: '%s', topic: '%s', consumerKey: '%s'", cfg.Exchange, cfg.Queue, cfg.RouteKey, consumerKey)
	<-forever
	return nil
}

func Subscriber(exchange string, exchangeKind ExchangeKind, queue, routeKey string, callback func(msg []byte) bool) error {
	return SubscriberWithConfig(SubscriberConfig{
		Exchange:     exchange,
		ExchangeKind: exchangeKind,
		Queue:        queue,
		RouteKey:     routeKey,
		ExitCount:    0,
		Callback:     callback,
	})
}
