package rabbitmq_outbound_adapter

import (
	"context"

	"go-boilerplate/internal/model"
	outbound_port "go-boilerplate/internal/port/outbound"
	"go-boilerplate/utils/rabbitmq"
)

type clientAdapter struct{}

func NewClientAdapter() outbound_port.ClientMessagePort {
	return &clientAdapter{}
}

func (adapter *clientAdapter) PublishUpsert(datas []model.ClientInput) error {
	err := rabbitmq.Publish(context.Background(), model.UpsertClientMessage, rabbitmq.KindFanOut, "", datas)
	if err != nil {
		return err
	}

	return nil
}
