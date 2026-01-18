package command_inbound_adapter

import (
	"context"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/activity"
	"prabogo/utils/log"
)

type clientAdapter struct {
	domain domain.Domain
}

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientCommandPort {
	return &clientAdapter{
		domain: domain,
	}
}

func (h *clientAdapter) PublishUpsert(name string) {
	ctx := activity.NewContext("command_client_publish_upsert")
	ctx = context.WithValue(ctx, activity.Payload, name)
	payload := []model.ClientInput{{Name: name}}

	err := h.domain.Client().PublishUpsert(ctx, payload)
	if err != nil {
		log.WithContext(ctx).Errorf("client publish upsert error %s: %s", err.Error(), name)
	}
	log.WithContext(ctx).Info("client publish upsert success")
}

func (h *clientAdapter) StartUpsert(name string) {
	ctx := activity.NewContext("command_client_start_upsert")
	ctx = context.WithValue(ctx, activity.Payload, name)
	payload := model.ClientInput{Name: name}

	err := h.domain.Client().StartUpsert(ctx, payload)
	if err != nil {
		log.WithContext(ctx).Errorf("client start upsert error %s: %s", err.Error(), name)
	}
	log.WithContext(ctx).Infof("client start upsert success")
}
