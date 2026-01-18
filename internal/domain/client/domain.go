package client

import (
	"context"

	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
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
	if len(inputs) == 0 {
		return nil, stacktrace.NewError("inputs is empty")
	}

	var filter model.ClientFilter
	for i := range inputs {
		model.ClientPrepare(&inputs[i])
		filter.Names = append(filter.Names, inputs[i].Name)
	}

	databaseClientPort := s.databasePort.Client()
	err := databaseClientPort.Upsert(inputs)
	if err != nil {
		return nil, stacktrace.Propagate(err, "upsert client error")
	}

	results, err := databaseClientPort.FindByFilter(filter, true)
	if err != nil {
		return nil, stacktrace.Propagate(err, "find client by filter error")
	}

	return results, nil
}

func (s *clientDomain) FindByFilter(ctx context.Context, filter model.ClientFilter) ([]model.Client, error) {
	if filter.IsEmpty() {
		return nil, stacktrace.NewError("filter is empty")
	}

	databaseClientPort := s.databasePort.Client()
	results, err := databaseClientPort.FindByFilter(filter, false)
	if err != nil {
		return nil, stacktrace.Propagate(err, "find client by filter error")
	}

	return results, nil
}

func (s *clientDomain) DeleteByFilter(ctx context.Context, filter model.ClientFilter) error {
	if filter.IsEmpty() {
		return stacktrace.NewError("filter is empty")
	}

	databaseClientPort := s.databasePort.Client()
	err := databaseClientPort.DeleteByFilter(filter)
	if err != nil {
		return stacktrace.Propagate(err, "delete client by filter error")
	}

	return nil
}

func (s *clientDomain) PublishUpsert(ctx context.Context, inputs []model.ClientInput) error {
	if len(inputs) == 0 {
		return stacktrace.NewError("inputs is empty")
	}

	messageClientPort := s.messagePort.Client()
	err := messageClientPort.PublishUpsert(inputs)
	if err != nil {
		return stacktrace.Propagate(err, "publish upsert client error")
	}

	return nil
}

func (s *clientDomain) IsExists(ctx context.Context, bearerKey string) (bool, error) {
	if bearerKey == "" {
		return false, stacktrace.NewError("bearerKey is empty")
	}

	var exists bool
	cacheClientPort := s.cachePort.Client()
	_, err := cacheClientPort.Get(bearerKey)
	if err != nil {
		if err == redis.Nil {
			databaseClientPort := s.databasePort.Client()
			exists, err = databaseClientPort.IsExists(bearerKey)
			if err != nil {
				return false, stacktrace.Propagate(err, "check if client exists error")
			}

			if exists {
				client, findErr := databaseClientPort.FindByFilter(model.ClientFilter{BearerKeys: []string{bearerKey}}, false)
				if findErr != nil {
					return false, stacktrace.Propagate(findErr, "find client by filter error")
				}

				if len(client) > 0 {
					setErr := cacheClientPort.Set(client[0])
					if setErr != nil {
						return false, stacktrace.Propagate(setErr, "set client to cache error")
					}
				}
			}
		} else {
			return false, stacktrace.Propagate(err, "get client from cache error")
		}
	} else {
		exists = true
	}

	return exists, nil
}

func (s *clientDomain) StartUpsert(ctx context.Context, input model.ClientInput) error {
	workflowClientPort := s.workflowPort.Client()
	return workflowClientPort.StartUpsert(input)
}
