package redis_outbound_adapter

import (
	"context"
	"encoding/json"

	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/redis"
	"prabogo/utils/tracing"
)

type clientAdapter struct{}

const (
	tracerName      = "outbound.cache"
	activityNameSet = "cache_client_set"
	activityNameGet = "cache_client_get"
)

func NewClientAdapter() outbound_port.ClientCachePort {
	return &clientAdapter{}
}

func (adapter *clientAdapter) Set(ctx context.Context, data model.Client) error {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameSet,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			semconv.DBSystemRedis,
			tracing.WithSpanAttributeKey("cache.operation", "set"),
		),
	)
	defer span.End()

	bytes, err := json.Marshal(data)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	err = redis.Set(ctx, data.BearerKey, string(bytes))
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func (adapter *clientAdapter) Get(ctx context.Context, bearerKey string) (model.Client, error) {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameGet,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithAttributes(
			semconv.DBSystemRedis,
			tracing.WithSpanAttributeKey("cache.operation", "get"),
		),
	)
	defer span.End()

	result, err := redis.Get(ctx, bearerKey)
	if err != nil {
		tracing.RecordError(span, err)
		return model.Client{}, err
	}

	var client model.Client
	err = json.Unmarshal([]byte(result), &client)
	if err != nil {
		tracing.RecordError(span, err)
		return model.Client{}, err
	}

	return client, nil
}
