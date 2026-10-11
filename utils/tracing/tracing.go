package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"prabogo/utils/activity"
)

type SpanOption func(*spanConfig)

type spanConfig struct {
	kind      trace.SpanKind
	attrs     []attribute.KeyValue
	semconvDB bool
	semconvMQ bool
	mqConsume bool
}

func WithSpanKind(k trace.SpanKind) SpanOption {
	return func(c *spanConfig) { c.kind = k }
}

func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(c *spanConfig) { c.attrs = append(c.attrs, attrs...) }
}

func WithDBSemconv() SpanOption {
	return func(c *spanConfig) { c.semconvDB = true }
}

func WithMQSemconv(consume bool) SpanOption {
	return func(c *spanConfig) { c.semconvMQ = true; c.mqConsume = consume }
}

func StartSpan(ctx context.Context, tracerName, name string, opts ...SpanOption) (context.Context, trace.Span) {
	cfg := spanConfig{kind: trace.SpanKindInternal}
	for _, o := range opts {
		o(&cfg)
	}

	attrs := cfg.attrs
	if cfg.semconvDB {
		attrs = append(attrs, semconv.DBSystemPostgreSQL)
	}
	if cfg.semconvMQ {
		attrs = append(attrs, semconv.MessagingSystemRabbitmq)
		if cfg.mqConsume {
			attrs = append(attrs, semconv.MessagingOperationTypeReceive)
		} else {
			attrs = append(attrs, semconv.MessagingOperationTypePublish)
		}
	}

	tracer := otel.Tracer(tracerName)
	spanCtx, span := tracer.Start(ctx, name,
		trace.WithSpanKind(cfg.kind),
		trace.WithAttributes(attrs...),
	)

	// Carry the activity fields onto the span as attributes for correlation.
	// Batch into a single SetAttributes call to minimize span lock acquisitions.
	activityAttrs := make([]attribute.KeyValue, 0, 3)
	if trxID, ok := activity.GetTransactionID(ctx); ok {
		activityAttrs = append(activityAttrs, attribute.String("activity.transaction_id", trxID))
	}
	if action, ok := activity.GetAction(ctx); ok {
		activityAttrs = append(activityAttrs, attribute.String("activity.action", action))
	}
	if clientID, ok := activity.GetClientID(ctx); ok {
		activityAttrs = append(activityAttrs, attribute.String("activity.client_id", clientID))
	}
	if len(activityAttrs) > 0 {
		span.SetAttributes(activityAttrs...)
	}

	return spanCtx, span
}

// toKeyValue converts a key/value pair into an attribute.KeyValue using a
// type switch. Unknown types fall back to a string representation via fmt.
func toKeyValue(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case bool:
		return attribute.Bool(key, v)
	case float64:
		return attribute.Float64(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

func WithSpanAttributeKey(key string, value interface{}) attribute.KeyValue {
	return toKeyValue(key, value)
}

func WithSpanAttribute(span trace.Span, key string, value interface{}) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(toKeyValue(key, value))
}

func RecordError(span trace.Span, err error) {
	if err != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
