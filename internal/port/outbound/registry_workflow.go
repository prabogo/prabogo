package outbound_port

//go:generate mockgen -source=registry_workflow.go -destination=./../../../tests/mocks/port/mock_registry_workflow.go
type WorkflowPort interface {
	Client() ClientWorkflowPort
}
