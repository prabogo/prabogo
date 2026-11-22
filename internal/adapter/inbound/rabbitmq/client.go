package rabbitmq_inbound_adapter

import (
	"context"
	"encoding/json"

	"github.com/palantir/stacktrace"
	"go.opentelemetry.io/otel/trace"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/activity"
	"prabogo/utils/log"
	"prabogo/utils/tracing"
)

type clientAdapter struct {
	domain domain.Domain
}

const (
	tracerName         = "inbound.message"
	activityNameUpsert = "message_client_upsert"
)

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientMessagePort {
	return &clientAdapter{
		domain: domain,
	}
}

func (h *clientAdapter) Upsert(ctx context.Context, a any) bool {
	msg := a.([]byte)
	ctx = activity.NewContext(activityNameUpsert, ctx)
	var payload []model.ClientInput
	err := json.Unmarshal(msg, &payload)
	if err != nil {
		log.WithContext(ctx).Errorf("message client upsert failed: %s", err.Error())
		return false
	}
	ctx = context.WithValue(ctx, activity.Payload, payload)

	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameUpsert,
		tracing.WithSpanKind(trace.SpanKindConsumer),
		tracing.WithMQSemconv(true),
		tracing.WithAttributes(tracing.WithSpanAttributeKey("messaging.destination.name", string(model.UpsertClientMessage))),
	)
	defer span.End()

	results, err := h.domain.Client().Upsert(ctx, payload)
	if err != nil {
		tracing.RecordError(span, err)
		log.WithContext(ctx).Errorf("message client upsert failed: %s", stacktrace.RootCause(err).Error())
		return false
	}
	ctx = context.WithValue(ctx, activity.Result, results)

	log.WithContext(ctx).Info("message client upsert success")
	return true
}
