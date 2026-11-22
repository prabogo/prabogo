package mcp_inbound_adapter

import (
	"context"

	"github.com/palantir/stacktrace"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/activity"
	"prabogo/utils/tracing"
)

type clientAdapter struct {
	domain domain.Domain
}

const (
	tracerName               = "inbound.mcp"
	activityNameFindByFilter = "mcp_client_find_by_filter"
)

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientMcpPort {
	return &clientAdapter{
		domain: domain,
	}
}

func (a *clientAdapter) Find(payload any) ([]model.Client, error) {
	ctx := activity.NewContext(activityNameFindByFilter)
	filter, ok := payload.(model.ClientFilter)
	if !ok {
		return nil, stacktrace.NewError("invalid payload type: expected model.ClientFilter")
	}

	ctx = context.WithValue(ctx, activity.Payload, filter)

	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameFindByFilter)
	defer span.End()

	results, err := a.domain.Client().FindByFilter(ctx, filter)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, stacktrace.Propagate(err, "find client by filter error")
	}

	return results, nil
}
