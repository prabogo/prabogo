package rabbitmq_outbound_adapter

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/rabbitmq"
	"prabogo/utils/tracing"
)

type clientAdapter struct{}

const (
	tracerName                = "outbound.message"
	activityNamePublishUpsert = "message_client_publish_upsert"
)

func NewClientAdapter() outbound_port.ClientMessagePort {
	return &clientAdapter{}
}

func (adapter *clientAdapter) PublishUpsert(ctx context.Context, datas []model.ClientInput) error {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNamePublishUpsert,
		tracing.WithSpanKind(trace.SpanKindProducer),
		tracing.WithMQSemconv(false),
		tracing.WithAttributes(
			tracing.WithSpanAttributeKey("message.count", len(datas)),
			tracing.WithSpanAttributeKey("messaging.destination.name", string(model.UpsertClientMessage)),
		),
	)
	defer span.End()

	err := rabbitmq.Publish(ctx, model.UpsertClientMessage, rabbitmq.KindFanOut, "", datas)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}
