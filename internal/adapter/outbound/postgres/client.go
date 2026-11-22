package postgres_outbound_adapter

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	"go.opentelemetry.io/otel/trace"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/tracing"
)

const tableClient = "clients"

const (
	tracerName         = "outbound.database"
	activityNameUpsert = "db_client_upsert"
	activityNameFind   = "db_client_find_by_filter"
	activityNameDelete = "db_client_delete_by_filter"
	activityNameExists = "db_client_is_exists"
)

type clientAdapter struct {
	db outbound_port.DatabaseExecutor
}

func NewClientAdapter(
	db outbound_port.DatabaseExecutor,
) outbound_port.ClientDatabasePort {
	return &clientAdapter{
		db: db,
	}
}

func (adapter *clientAdapter) Upsert(ctx context.Context, datas []model.ClientInput) error {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameUpsert,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithDBSemconv(),
		tracing.WithAttributes(tracing.WithSpanAttributeKey("client.count", len(datas))),
	)
	defer span.End()

	dataset := goqu.Dialect("postgres").
		Insert(tableClient).
		Rows(datas)

	query, _, err := dataset.ToSQL()
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	query += ` ON CONFLICT (bearer_key) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at`
	_, err = adapter.db.ExecContext(ctx, query)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	return nil
}

func (adapter *clientAdapter) FindByFilter(ctx context.Context, filter model.ClientFilter, lock bool) (result []model.Client, err error) {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameFind,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithDBSemconv(),
		tracing.WithAttributes(tracing.WithSpanAttributeKey("db.lock", lock)),
	)
	defer span.End()

	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient)
	dataset = addFilter(dataset, filter)

	query, _, err := dataset.ToSQL()
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}

	if lock {
		query += " FOR UPDATE"
	}

	res, err := adapter.db.QueryContext(ctx, query)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}
	defer res.Close()

	clients := []model.Client{}
	for res.Next() {
		result := model.Client{}
		err := res.Scan(
			&result.ID,
			&result.Name,
			&result.BearerKey,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
		if err != nil {
			tracing.RecordError(span, err)
			return nil, err
		}

		clients = append(clients, result)
	}

	tracing.WithSpanAttribute(span, "db.rows_returned", len(clients))
	return clients, nil
}

func (adapter *clientAdapter) DeleteByFilter(ctx context.Context, filter model.ClientFilter) error {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameDelete,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithDBSemconv(),
	)
	defer span.End()

	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient)
	dataset = addFilter(dataset, filter)

	query, _, err := dataset.Delete().ToSQL()
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	_, err = adapter.db.ExecContext(ctx, query)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	return nil
}

func (adapter *clientAdapter) IsExists(ctx context.Context, bearerKey string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, tracerName, activityNameExists,
		tracing.WithSpanKind(trace.SpanKindClient),
		tracing.WithDBSemconv(),
	)
	defer span.End()

	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient).Select("id").Where(goqu.Ex{"bearer_key": bearerKey}).Limit(1)

	query, _, err := dataset.ToSQL()
	if err != nil {
		tracing.RecordError(span, err)
		return false, err
	}

	// Use QueryRowContext instead of QueryContext: an existence check needs at
	// most one row, and it avoids allocating a *sql.Rows cursor.
	var id int
	err = adapter.db.QueryRowContext(ctx, query).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		tracing.RecordError(span, err)
		return false, err
	}

	return true, nil
}

func addFilter(dataset *goqu.SelectDataset, filter model.ClientFilter) *goqu.SelectDataset {
	if filter.IDs != nil {
		dataset = dataset.Where(goqu.Ex{"id": filter.IDs})
	}

	if filter.Names != nil {
		dataset = dataset.Where(goqu.Ex{"name": filter.Names})
	}

	if filter.BearerKeys != nil {
		dataset = dataset.Where(goqu.Ex{"bearer_key": filter.BearerKeys})
	}

	return dataset
}
