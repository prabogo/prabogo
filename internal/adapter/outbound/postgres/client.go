package postgres_outbound_adapter

import (
	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

const tableClient = "clients"

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

func (adapter *clientAdapter) Upsert(datas []model.ClientInput) error {
	dataset := goqu.Dialect("postgres").
		Insert(tableClient).
		Rows(datas)

	query, _, err := dataset.ToSQL()
	if err != nil {
		return err
	}

	query += ` ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at`
	_, err = adapter.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (adapter *clientAdapter) FindByFilter(filter model.ClientFilter, lock bool) (result []model.Client, err error) {
	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient)
	dataset = addFilter(dataset, filter)

	query, _, err := dataset.ToSQL()
	if err != nil {
		return nil, err
	}

	if lock {
		query += " FOR UPDATE"
	}

	res, err := adapter.db.Query(query)
	if err != nil {
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
			return nil, err
		}

		clients = append(clients, result)
	}

	return clients, nil
}

func (adapter *clientAdapter) DeleteByFilter(filter model.ClientFilter) error {
	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient)
	dataset = addFilter(dataset, filter)

	query, _, err := dataset.Delete().ToSQL()
	if err != nil {
		return err
	}

	res, err := adapter.db.Query(query)
	if err != nil {
		return err
	}
	defer res.Close()

	return nil
}

func (adapter *clientAdapter) IsExists(bearerKey string) (bool, error) {
	dialect := goqu.Dialect("postgres")
	dataset := dialect.From(tableClient).Select("id").Where(goqu.Ex{"bearer_key": bearerKey})

	query, _, err := dataset.ToSQL()
	if err != nil {
		return false, err
	}

	res, err := adapter.db.Query(query)
	if err != nil {
		return false, err
	}
	defer res.Close()

	return res.Next(), nil
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
