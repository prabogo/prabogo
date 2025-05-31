package rabbitmq_outbound_adapter

import (
	outbound_port "go-boilerplate/internal/port/outbound"
)

type adapter struct {
}

func NewAdapter() outbound_port.MessagePort {
	return &adapter{}
}

func (s *adapter) Client() outbound_port.ClientMessagePort {
	return NewClientAdapter()
}
