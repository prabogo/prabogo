package inbound_port

import (
	"context"

	"prabogo/internal/model"
)

type ClientHttpPort interface {
	Upsert(a any) error
	Find(a any) error
	Delete(a any) error
}

type ClientMessagePort interface {
	Upsert(ctx context.Context, a any) bool
}

type ClientCommandPort interface {
	PublishUpsert(name string)
	StartUpsert(name string)
}

type ClientWorkflowPort interface {
	Upsert()
}

type ClientMcpPort interface {
	Find(a any) ([]model.Client, error)
}
