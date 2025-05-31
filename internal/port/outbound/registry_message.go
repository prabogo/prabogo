package outbound_port

//go:generate mockgen -source=registry_message.go -destination=./../../../tests/mocks/port/mock_registry_message.go
type MessagePort interface {
	Client() ClientMessagePort
}
