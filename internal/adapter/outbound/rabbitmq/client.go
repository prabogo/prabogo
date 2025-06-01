package rabbitmq_outbound_adapter

import (
	"context"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/rabbitmq"
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
