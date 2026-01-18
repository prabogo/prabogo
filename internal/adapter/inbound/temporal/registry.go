package temporal_inbound_adapter

import (
	client_temporal_inbound_adapter "prabogo/internal/adapter/inbound/temporal/client"
	"prabogo/internal/domain"
	inbound_port "prabogo/internal/port/inbound"
)

type adapter struct {
	domain domain.Domain
}

func NewAdapter(
	domain domain.Domain,
) inbound_port.WorkflowPort {
	return &adapter{
		domain: domain,
	}
}

func (a *adapter) Client() inbound_port.ClientWorkflowPort {
	return client_temporal_inbound_adapter.NewClientAdapter(a.domain)
}
