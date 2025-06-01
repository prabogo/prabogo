package rabbitmq_inbound_adapter

import (
	"prabogo/internal/domain"
	inbound_port "prabogo/internal/port/inbound"
)

type adapter struct {
	domain domain.Domain
}

func NewAdapter(
	domain domain.Domain,
) inbound_port.MessagePort {
	return &adapter{
		domain: domain,
	}
}

func (a *adapter) Client() inbound_port.ClientMessagePort {
	return NewClientAdapter(a.domain)
}
