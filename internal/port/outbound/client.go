package outbound_port

import (
	"context"

	"prabogo/internal/model"
)

//go:generate mockgen -source=client.go -destination=./../../../tests/mocks/port/mock_client.go
type ClientDatabasePort interface {
	Upsert(ctx context.Context, datas []model.ClientInput) error
	FindByFilter(ctx context.Context, filter model.ClientFilter, lock bool) ([]model.Client, error)
	DeleteByFilter(ctx context.Context, filter model.ClientFilter) error
	IsExists(ctx context.Context, bearerKey string) (bool, error)
}

type ClientMessagePort interface {
	PublishUpsert(ctx context.Context, datas []model.ClientInput) error
}

type ClientCachePort interface {
	Set(ctx context.Context, data model.Client) error
	Get(ctx context.Context, bearerKey string) (model.Client, error)
}

type ClientWorkflowPort interface {
	StartUpsert(ctx context.Context, data model.ClientInput) error
}
