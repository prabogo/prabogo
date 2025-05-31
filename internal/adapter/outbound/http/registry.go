package http_outbound_adapter

import outbound_port "go-boilerplate/internal/port/outbound"

type adapter struct{}

func NewAdapter() outbound_port.HttpPort {
	return &adapter{}
}
