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
}

func NewDomain(
	databasePort outbound_port.DatabasePort,
	messagePort outbound_port.MessagePort,
	cachePort outbound_port.CachePort,
) Domain {
	return &domain{
		databasePort: databasePort,
		messagePort:  messagePort,
		cachePort:    cachePort,
	}
}

func (d *domain) Client() client.ClientDomain {
	return client.NewClientDomain(d.databasePort, d.messagePort, d.cachePort)
}
