package inbound_port

type PingHttpPort interface {
	GetResource(a any) error
}
