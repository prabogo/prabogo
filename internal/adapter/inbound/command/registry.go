package command_inbound_adapter

import (
	"prabogo/internal/domain"
	inbound_port "prabogo/internal/port/inbound"
)

type adapter struct {
	domain domain.Domain
}

func NewAdapter(
	domain domain.Domain,
) inbound_port.CommandPort {
	return &adapter{
		domain: domain,
	}
}

func (s *adapter) Client() inbound_port.ClientCommandPort {
	return NewClientAdapter(s.domain)
}
