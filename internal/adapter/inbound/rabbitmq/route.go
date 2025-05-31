package rabbitmq_inbound_adapter

import (
	"context"
	"os"

	"go-boilerplate/internal/model"
	inbound_port "go-boilerplate/internal/port/inbound"
	"go-boilerplate/utils/log"
	"go-boilerplate/utils/rabbitmq"
)

func InitRoute(
	ctx context.Context,
	args []string,
	port inbound_port.MessagePort,
) {
	if len(args) > 2 {
		switch args[2] {
		case "upsert_client":
			log.WithContext(ctx).Info("message subscribe upsert client started")
			done := make(chan struct{})
			go func() {
				err := rabbitmq.Subscriber(
					model.UpsertClientMessage,
					rabbitmq.KindFanOut,
					os.Getenv("UPSERT_CLIENT_MESSAGE_SUBSCRIBE"),
					"",
					func(msg []byte) bool {
						return port.Client().Upsert(msg)
					},
				)
				if err != nil {
					log.WithContext(ctx).Errorf("failed to subscribe to %s: %s", model.UpsertClientMessage, err)
				}
				close(done)
			}()
			<-done
		default:
			log.WithContext(ctx).Info("message subscribe not found")
		}
	} else {
		log.WithContext(ctx).Info("message subscribe not found")
	}
}
