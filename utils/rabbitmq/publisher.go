package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

//go:generate mockgen -source=publisher.go -destination=./../../tests/mocks/mock_utils/mock_rabbitmq/mock_publisher.go
type Publisher interface {
	Publish(ctx context.Context, exchange string, exchangeKind ExchangeKind, routeKey string, msg any) error
}

func NewPublisher() Publisher {
	return &publisher{}
}

type publisher struct{}

func (p *publisher) Publish(ctx context.Context, exchange string, exchangeKind ExchangeKind, routeKey string, msg any) error {
	return Publish(ctx, exchange, exchangeKind, routeKey, msg)
}

func Publish(ctx context.Context, exchange string, exchangeKind ExchangeKind, routeKey string, msg any) (err error) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Use global connection from InitMessage (singleton)
	if initErr := InitMessage(); initErr != nil {
		return initErr
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
		exchange,
		string(exchangeKind),
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		ctx,
		exchange,
		routeKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgBytes,
		})
	if err != nil {
		return err
	}

	return nil
}
