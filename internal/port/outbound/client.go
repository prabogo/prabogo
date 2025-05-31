package outbound_port

import "go-boilerplate/internal/model"

//go:generate mockgen -source=client.go -destination=./../../../tests/mocks/port/mock_client.go
type ClientDatabasePort interface {
	Upsert(datas []model.ClientInput) error
	FindByFilter(filter model.ClientFilter, lock bool) ([]model.Client, error)
	DeleteByFilter(filter model.ClientFilter) error
	IsExists(bearerKey string) (bool, error)
}

type ClientMessagePort interface {
	PublishUpsert(datas []model.ClientInput) error
}

type ClientCachePort interface {
	Set(data model.Client) error
	Get(bearerKey string) (model.Client, error)
}
