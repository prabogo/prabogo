package outbound_port

//go:generate mockgen -source=registry_cache.go -destination=./../../../tests/mocks/port/mock_registry_cache.go
type CachePort interface {
	Client() ClientCachePort
}
