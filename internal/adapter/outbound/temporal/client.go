package temporal_outbound_adapter

import (
	"context"
	"os"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/temporal"
	"prabogo/utils/tracing"
)

type clientWorkflowAdapter struct{}

const (
	tracerName              = "outbound.workflow"
	activityNameStartUpsert = "workflow_client_start_upsert"
)

func NewClientWorkflowAdapter() outbound_port.ClientWorkflowPort {
	return &clientWorkflowAdapter{}
}

func (g *clientWorkflowAdapter) StartUpsert(ctx context.Context, input model.ClientInput) error {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameStartUpsert,
		tracing.WithAttributes(tracing.WithSpanAttributeKey("domain.operation", "start_upsert")),
	)
	defer span.End()

	namespace := os.Getenv("WORKFLOW_NAMESPACE")
	_, err := temporal.ExecuteWorkflow(ctx, namespace, model.UpsertClientWorkflowName, input)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	return nil
}
