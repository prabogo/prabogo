package outbound_port

import (
	"context"
	"database/sql"
)

//go:generate mockgen -source=registry_database.go -destination=./../../../tests/mocks/port/mock_registry_database.go
type InTransaction func(repoRegistry DatabasePort) (interface{}, error)

type DatabasePort interface {
	Client() ClientDatabasePort
	DoInTransaction(txFunc InTransaction) (out interface{}, err error)
}

type DatabaseExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
