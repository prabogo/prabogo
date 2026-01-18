package temporal

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/types/known/durationpb"
)

type WorkerConfig struct {
	MaxConcurrentActivityTasks   int
	MaxConcurrentWorkflowTasks   int
	MaxConcurrentLocalActivities int
	TaskQueueActivitiesPerSecond float64
}

func NewWorker(ctx context.Context, name string) (worker.Worker, error) {
	hostPort := fmt.Sprintf("%s:%s", os.Getenv("WORKFLOW_HOST"), os.Getenv("WORKFLOW_PORT"))
	namespace := getNamespace()

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

	w := worker.New(c, name, worker.Options{})

	return w, nil
}

// getNamespace returns the namespace with fallback to default
func getNamespace() string {
	namespace := os.Getenv("WORKFLOW_NAMESPACE")
	if namespace == "" {
		namespace = "default" // Fallback to default namespace
	}
	return namespace
}

// ensureNamespaceExists creates namespace if it doesn't exist
func ensureNamespaceExists(ctx context.Context, hostPort, namespace string) error {
	// Skip namespace creation for default namespace (always exists)
	if namespace == "default" {
		return nil
	}

	// Create namespace client without namespace to manage namespaces
	nc, err := client.NewNamespaceClient(client.Options{
		HostPort: hostPort,
		// Don't set namespace here - we're managing namespaces
	})
	if err != nil {
		return fmt.Errorf("failed to create namespace client: %w", err)
	}
	defer nc.Close()

	// Check if namespace exists
	_, err = nc.Describe(ctx, namespace)
	if err == nil {
		// Namespace exists, no need to create
		return nil
	}

	// Create context with timeout for namespace creation
	createCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Namespace doesn't exist, create it
	err = nc.Register(createCtx, &workflowservice.RegisterNamespaceRequest{
		Namespace: namespace,
		WorkflowExecutionRetentionPeriod: &durationpb.Duration{
			Seconds: 7 * 24 * 60 * 60, // 7 days retention for channel integration
		},
		Description: fmt.Sprintf("Channel Integration namespace for %s marketplace operations", namespace),
	})
	if err != nil {
		return fmt.Errorf("failed to register namespace %s: %w", namespace, err)
	}

	// Wait a moment for namespace to be ready
	time.Sleep(2 * time.Second)

	return nil
}
