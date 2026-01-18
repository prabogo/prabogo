package inbound_port

type WorkflowPort interface {
	Client() ClientWorkflowPort
}
