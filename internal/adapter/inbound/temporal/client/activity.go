package client_temporal_inbound_adapter

import (
	"context"

	"go.temporal.io/sdk/temporal"

	"prabogo/internal/domain"
	"prabogo/internal/model"
)

type ClientActivity interface {
	Upsert(ctx context.Context, input []model.ClientInput) ([]model.Client, error)
}

type clientActivity struct {
	domain domain.Domain
}

func NewClientActivity(
	domain domain.Domain,
) ClientActivity {
	return &clientActivity{
		domain: domain,
	}
}

func (a *clientActivity) Upsert(ctx context.Context, input []model.ClientInput) ([]model.Client, error) {
	result, err := a.domain.Client().Upsert(ctx, input)
	if err != nil {
		if err.Error() == "inputs is empty" {
			return nil, temporal.NewNonRetryableApplicationError(err.Error(), "ClientNotFound", err)
		}
		return nil, err
	}
	return result, nil
}
