package temporal

import (
	"context"
	"fmt"
	"os"

	"github.com/pborman/uuid"
	"go.temporal.io/sdk/client"
)

func ExecuteWorkflow(ctx context.Context, namespace, name string, input interface{}) (client.WorkflowRun, error) {
	hostPort := fmt.Sprintf("%s:%s", os.Getenv("WORKFLOW_HOST"), os.Getenv("WORKFLOW_PORT"))

	// Ensure namespace exists with proper error handling
	err := ensureNamespaceExists(ctx, hostPort, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	c, err := client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial temporal client: %w", err)
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        name + uuid.New(),
		TaskQueue: name,
	}

	return c.ExecuteWorkflow(ctx, workflowOptions, name, input)
}
