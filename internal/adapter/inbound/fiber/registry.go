package fiber_inbound_adapter

import (
	"go-boilerplate/internal/domain"
	inbound_port "go-boilerplate/internal/port/inbound"
)

type adapter struct {
	domain domain.Domain
}

func NewAdapter(
	domain domain.Domain,
) inbound_port.HttpPort {
	return &adapter{
		domain: domain,
	}
}

func (s *adapter) Ping() inbound_port.PingHttpPort {
	return NewPingAdapter()
}

func (s *adapter) Middleware() inbound_port.MiddlewareHttpPort {
	return NewMiddlewareAdapter(s.domain)
}

func (s *adapter) Client() inbound_port.ClientHttpPort {
	return NewClientAdapter(s.domain)
}
