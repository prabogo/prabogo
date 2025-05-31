package client

import (
	"context"

	"github.com/palantir/stacktrace"
	"github.com/redis/go-redis/v9"

	"go-boilerplate/internal/model"
	outbound_port "go-boilerplate/internal/port/outbound"
)

type ClientDomain interface {
	Upsert(ctx context.Context, inputs []model.ClientInput) ([]model.Client, error)
	FindByFilter(ctx context.Context, filter model.ClientFilter) ([]model.Client, error)
	DeleteByFilter(ctx context.Context, filter model.ClientFilter) error
	PublishUpsert(ctx context.Context, inputs []model.ClientInput) error
	IsExists(ctx context.Context, bearerKey string) (bool, error)
}

type clientDomain struct {
	databasePort outbound_port.DatabasePort
	messagePort  outbound_port.MessagePort
	cachePort    outbound_port.CachePort
}

func NewClientDomain(
	databasePort outbound_port.DatabasePort,
	messagePort outbound_port.MessagePort,
	cachePort outbound_port.CachePort,
) ClientDomain {
	return &clientDomain{
		databasePort: databasePort,
		messagePort:  messagePort,
		cachePort:    cachePort,
	}
}

func (s *clientDomain) Upsert(ctx context.Context, inputs []model.ClientInput) ([]model.Client, error) {
	if len(inputs) == 0 {
		return nil, stacktrace.NewError("inputs is empty")
	}

	var filter model.ClientFilter
	for i := range inputs {
		model.ClientPrepare(&inputs[i])
		filter.BearerKeys = append(filter.BearerKeys, inputs[i].BearerKey)
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

			client, err := databaseClientPort.FindByFilter(model.ClientFilter{BearerKeys: []string{bearerKey}}, false)
			if err != nil {
				return false, stacktrace.Propagate(err, "find client by filter error")
			}

			if len(client) > 0 {
				err = cacheClientPort.Set(client[0])
				if err != nil {
					return false, stacktrace.Propagate(err, "set client to cache error")
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
