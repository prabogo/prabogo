package domain

import (
	"prabogo/internal/domain/client"
	outbound_port "prabogo/internal/port/outbound"
)

type Domain interface {
	Client() client.ClientDomain
}

type domain struct {
	databasePort outbound_port.DatabasePort
	messagePort  outbound_port.MessagePort
	cachePort    outbound_port.CachePort
	workflowPort outbound_port.WorkflowPort
}

func NewDomain(
	databasePort outbound_port.DatabasePort,
	messagePort outbound_port.MessagePort,
	cachePort outbound_port.CachePort,
	workflowPort outbound_port.WorkflowPort,
) Domain {
	return &domain{
		databasePort: databasePort,
		messagePort:  messagePort,
		cachePort:    cachePort,
		workflowPort: workflowPort,
	}
}

func (d *domain) Client() client.ClientDomain {
	return client.NewClientDomain(d.databasePort, d.messagePort, d.cachePort, d.workflowPort)
}
