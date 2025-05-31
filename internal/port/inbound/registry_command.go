package inbound_port

type CommandPort interface {
	Client() ClientCommandPort
}
