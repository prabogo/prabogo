package temporal_outbound_adapter

import (
	outbound_port "prabogo/internal/port/outbound"
)

type adapter struct{}

func NewAdapter() outbound_port.WorkflowPort {
	return &adapter{}
}

func (a *adapter) Client() outbound_port.ClientWorkflowPort {
	return NewClientWorkflowAdapter()
}
