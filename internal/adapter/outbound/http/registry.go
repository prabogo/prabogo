package http_outbound_adapter

import outbound_port "prabogo/internal/port/outbound"

type adapter struct{}

func NewAdapter() outbound_port.HttpPort {
	return &adapter{}
}
