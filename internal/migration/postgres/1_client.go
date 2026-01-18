package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upClient, downClient)
}

func upClient(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS clients (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) UNIQUE NOT NULL,
		bearer_key VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
	);`)
	if err != nil {
		return err
	}
	return nil
}

func downClient(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec(`DROP TABLE clients;`)
	if err != nil {
		return err
	}
	return nil
}
