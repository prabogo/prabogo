package client_temporal_inbound_adapter

import (
	"go.temporal.io/sdk/worker"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	inbound_port "prabogo/internal/port/inbound"
	"prabogo/utils/activity"
	"prabogo/utils/log"
	"prabogo/utils/temporal"
	"prabogo/utils/tracing"
)

type clientAdapter struct {
	domain domain.Domain
}

const (
	tracerNameWorker        = "inbound.workflow"
	activityNameStartWorker = "upsert_client_worker"
)

func NewClientAdapter(
	domain domain.Domain,
) inbound_port.ClientWorkflowPort {
	return &clientAdapter{
		domain: domain,
	}
}

func (a *clientAdapter) Upsert() {
	ctx := activity.NewContext(activityNameStartWorker)

	ctx, span := tracing.StartSpan(ctx, tracerNameWorker, activityNameStartWorker)
	defer span.End()

	w, err := temporal.NewWorker(ctx, model.UpsertClientWorkflowName)
	if err != nil {
		tracing.RecordError(span, err)
		log.WithContext(ctx).Error("unable to create worker", err)
		return
	}

	activity := NewClientActivity(a.domain)
	workflow := NewClientWorkflow(activity)

	w.RegisterWorkflow(workflow.UpsertClientWorkflow)
	w.RegisterActivity(activity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		tracing.RecordError(span, err)
		log.WithContext(ctx).Error("unable to start worker", err)
		return
	}
}
