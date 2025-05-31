package outbound_port

//go:generate mockgen -source=registry_http.go -destination=./../../../tests/mocks/port/mock_registry_http.go
type HttpPort interface {
}
