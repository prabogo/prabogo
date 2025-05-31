package inbound_port

type ClientHttpPort interface {
	Upsert(a any) error
	Find(a any) error
	Delete(a any) error
}

type ClientMessagePort interface {
	Upsert(a any) bool
}

type ClientCommandPort interface {
	PublishUpsert(name string)
}
