package redis_outbound_adapter

import (
	outbound_port "prabogo/internal/port/outbound"
)

type adapter struct {
}

func NewAdapter() outbound_port.CachePort {
	return &adapter{}
}

func (s *adapter) Client() outbound_port.ClientCachePort {
	return NewClientAdapter()
}
