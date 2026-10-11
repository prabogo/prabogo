package client

import (
	"context"

	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/tracing"
)

type ClientDomain interface {
	Upsert(ctx context.Context, inputs []model.ClientInput) ([]model.Client, error)
	FindByFilter(ctx context.Context, filter model.ClientFilter) ([]model.Client, error)
	DeleteByFilter(ctx context.Context, filter model.ClientFilter) error
	PublishUpsert(ctx context.Context, inputs []model.ClientInput) error
	IsExists(ctx context.Context, bearerKey string) (bool, error)
	StartUpsert(ctx context.Context, input model.ClientInput) error
}

type clientDomain struct {
	databasePort outbound_port.DatabasePort
	messagePort  outbound_port.MessagePort
	cachePort    outbound_port.CachePort
	workflowPort outbound_port.WorkflowPort
}

func NewClientDomain(
	databasePort outbound_port.DatabasePort,
	messagePort outbound_port.MessagePort,
	cachePort outbound_port.CachePort,
	workflowPort outbound_port.WorkflowPort,
) ClientDomain {
	return &clientDomain{
		databasePort: databasePort,
		messagePort:  messagePort,
		cachePort:    cachePort,
		workflowPort: workflowPort,
	}
}

func (s *clientDomain) Upsert(ctx context.Context, inputs []model.ClientInput) ([]model.Client, error) {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.upsert",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "upsert")),
	)
	defer span.End()

	if len(inputs) == 0 {
		err := stacktrace.NewError("inputs is empty")
		tracing.RecordError(span, err)
		return nil, err
	}

	var filter model.ClientFilter
	for i := range inputs {
		model.ClientPrepare(&inputs[i])
		filter.Names = append(filter.Names, inputs[i].Name)
	}
	tracing.WithSpanAttribute(span, "client.count", len(inputs))

	databaseClientPort := s.databasePort.Client()
	err := databaseClientPort.Upsert(ctx, inputs)
	if err != nil {
		err = stacktrace.Propagate(err, "upsert client error")
		tracing.RecordError(span, err)
		return nil, err
	}

	results, err := databaseClientPort.FindByFilter(ctx, filter, true)
	if err != nil {
		err = stacktrace.Propagate(err, "find client by filter error")
		tracing.RecordError(span, err)
		return nil, err
	}

	return results, nil
}

func (s *clientDomain) FindByFilter(ctx context.Context, filter model.ClientFilter) ([]model.Client, error) {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.find_by_filter",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "find_by_filter")),
	)
	defer span.End()

	if filter.IsEmpty() {
		err := stacktrace.NewError("filter is empty")
		tracing.RecordError(span, err)
		return nil, err
	}

	databaseClientPort := s.databasePort.Client()
	results, err := databaseClientPort.FindByFilter(ctx, filter, false)
	if err != nil {
		err = stacktrace.Propagate(err, "find client by filter error")
		tracing.RecordError(span, err)
		return nil, err
	}

	return results, nil
}

func (s *clientDomain) DeleteByFilter(ctx context.Context, filter model.ClientFilter) error {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.delete_by_filter",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "delete_by_filter")),
	)
	defer span.End()

	if filter.IsEmpty() {
		err := stacktrace.NewError("filter is empty")
		tracing.RecordError(span, err)
		return err
	}

	databaseClientPort := s.databasePort.Client()
	err := databaseClientPort.DeleteByFilter(ctx, filter)
	if err != nil {
		err = stacktrace.Propagate(err, "delete client by filter error")
		tracing.RecordError(span, err)
		return err
	}

	return nil
}

func (s *clientDomain) PublishUpsert(ctx context.Context, inputs []model.ClientInput) error {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.publish_upsert",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "publish_upsert")),
	)
	defer span.End()

	if len(inputs) == 0 {
		err := stacktrace.NewError("inputs is empty")
		tracing.RecordError(span, err)
		return err
	}

	messageClientPort := s.messagePort.Client()
	err := messageClientPort.PublishUpsert(ctx, inputs)
	if err != nil {
		err = stacktrace.Propagate(err, "publish upsert client error")
		tracing.RecordError(span, err)
		return err
	}

	return nil
}

func (s *clientDomain) IsExists(ctx context.Context, bearerKey string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.is_exists",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "is_exists")),
	)
	defer span.End()

	if bearerKey == "" {
		err := stacktrace.NewError("bearerKey is empty")
		tracing.RecordError(span, err)
		return false, err
	}

	var exists bool
	cacheClientPort := s.cachePort.Client()
	_, err := cacheClientPort.Get(ctx, bearerKey)
	if err != nil {
		if err == redis.Nil {
			databaseClientPort := s.databasePort.Client()
			exists, err = databaseClientPort.IsExists(ctx, bearerKey)
			if err != nil {
				err = stacktrace.Propagate(err, "check if client exists error")
				tracing.RecordError(span, err)
				return false, err
			}

			if exists {
				client, findErr := databaseClientPort.FindByFilter(ctx, model.ClientFilter{BearerKeys: []string{bearerKey}}, false)
				if findErr != nil {
					findErr = stacktrace.Propagate(findErr, "find client by filter error")
					tracing.RecordError(span, findErr)
					return false, findErr
				}

				if len(client) > 0 {
					setErr := cacheClientPort.Set(ctx, client[0])
					if setErr != nil {
						setErr = stacktrace.Propagate(setErr, "set client to cache error")
						tracing.RecordError(span, setErr)
						return false, setErr
					}
				}
			}
		} else {
			err = stacktrace.Propagate(err, "get client from cache error")
			tracing.RecordError(span, err)
			return false, err
		}
	} else {
		exists = true
	}

	return exists, nil
}

func (s *clientDomain) StartUpsert(ctx context.Context, input model.ClientInput) error {
	ctx, span := tracing.StartSpan(ctx, "domain", "client.start_upsert",
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "start_upsert")),
	)
	defer span.End()

	workflowClientPort := s.workflowPort.Client()
	err := workflowClientPort.StartUpsert(ctx, input)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}
