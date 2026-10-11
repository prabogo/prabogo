package client_temporal_inbound_adapter

import (
	"context"

	"go.temporal.io/sdk/temporal"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	"prabogo/utils/activity"
	"prabogo/utils/log"
	"prabogo/utils/tracing"
)

type ClientActivity interface {
	Upsert(ctx context.Context, input []model.ClientInput) ([]model.Client, error)
}

type clientActivity struct {
	domain domain.Domain
}

const (
	tracerNameActivity     = "inbound.workflow"
	activityNameUpsert     = "workflow_client_upsert_activity"
)

func NewClientActivity(
	domain domain.Domain,
) ClientActivity {
	return &clientActivity{
		domain: domain,
	}
}

func (a *clientActivity) Upsert(ctx context.Context, input []model.ClientInput) ([]model.Client, error) {
	ctx = activity.NewContext(activityNameUpsert, ctx)
	ctx = context.WithValue(ctx, activity.Payload, input)

	ctx, span := tracing.StartSpan(ctx, tracerNameActivity, activityNameUpsert)
	defer span.End()

	result, err := a.domain.Client().Upsert(ctx, input)
	if err != nil {
		tracing.RecordError(span, err)
		if err.Error() == "inputs is empty" {
			return nil, temporal.NewNonRetryableApplicationError(err.Error(), "ClientNotFound", err)
		}
		return nil, err
	}

	ctx = context.WithValue(ctx, activity.Result, result)
	log.WithContext(ctx).Info("workflow client upsert activity success")
	return result, nil
}
