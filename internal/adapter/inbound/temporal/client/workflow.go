package client_temporal_inbound_adapter

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"prabogo/internal/domain"
	"prabogo/internal/model"
)

type ClientWorkflow interface {
	UpsertClientWorkflow(ctx workflow.Context, input model.ClientInput) (string, error)
}

type clientWorkflow struct {
	domain domain.Domain
}

func NewClientWorkflow(
	domain domain.Domain,
) ClientWorkflow {
	return &clientWorkflow{
		domain: domain,
	}
}

func (g *clientWorkflow) UpsertClientWorkflow(ctx workflow.Context, input model.ClientInput) (string, error) {
	logger := workflow.GetLogger(ctx)
	workflowInfo := workflow.GetInfo(ctx)

	logger.Info("Workflow started", "WorkflowID", workflowInfo.WorkflowExecution.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var results []model.Client
	err := workflow.ExecuteActivity(
		ctx,
		g.domain.Client().Upsert,
		[]model.ClientInput{input},
	).Get(ctx, &results)
	if err != nil {
		logger.Error("UpsertClient activity failed", "Error", err)
		return "Failed to upsert client", err
	}

	var bearerKey string
	if len(results) > 0 {
		bearerKey = results[0].BearerKey
	}

	successMessage := "Bearer key: " + bearerKey
	logger.Info(successMessage, "WorkflowID", workflowInfo.WorkflowExecution.ID)

	return successMessage, nil
}
