package temporal_outbound_adapter

import (
	"context"
	"os"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/temporal"
)

type clientWorkflowAdapter struct{}

func NewClientWorkflowAdapter() outbound_port.ClientWorkflowPort {
	return &clientWorkflowAdapter{}
}

func (g *clientWorkflowAdapter) StartUpsert(input model.ClientInput) error {
	namespace := os.Getenv("WORKFLOW_NAMESPACE")
	_, err := temporal.ExecuteWorkflow(context.Background(), namespace, model.UpsertClientWorkflowName, input)
	if err != nil {
		return err
	}

	return nil
}
