package rabbitmq_inbound_adapter

import (
	"context"
	"encoding/json"
	"go-boilerplate/internal/domain"
	"go-boilerplate/internal/model"
	inbound_port "go-boilerplate/internal/port/inbound"
	"go-boilerplate/utils/activity"
	"go-boilerplate/utils/log"
)

type clientAdapter struct {
	domain domain.Domain
}

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientMessagePort {
	return &clientAdapter{
		domain: domain,
	}
}

func (h *clientAdapter) Upsert(a any) bool {
	msg := a.([]byte)
	ctx := activity.NewContext("message_client_upsert")
	var payload []model.ClientInput
	err := json.Unmarshal(msg, &payload)
	if err != nil {
		log.WithContext(ctx).Errorf("client upsert error %s: %s", err.Error(), string(msg))
		return true
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	results, err := h.domain.Client().Upsert(ctx, payload)
	if err != nil {
		log.WithContext(ctx).Errorf("client upsert error %s: %s", err.Error(), string(msg))
	}
	ctx = context.WithValue(ctx, activity.Result, results)

	log.WithContext(ctx).Info("client upsert success")
	return true
}
