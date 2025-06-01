package database

import (
	"context"
	"database/sql"
	"os"

	"github.com/pressly/goose/v3"

	"prabogo/utils"
	"prabogo/utils/log"
)

func InitDatabase(ctx context.Context, outboundDatabaseDriver string) *sql.DB {
	db, err := sql.Open(outboundDatabaseDriver, utils.GetDatabaseString())
	if err != nil {
		log.WithContext(ctx).Fatalf("failed to open database: %+v", err)
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		log.WithContext(ctx).Fatalf("failed to connect database: %+v", err)
		os.Exit(1)
	}

	if err := goose.Up(db, utils.GetMigrationDir()); err != nil {
		log.WithContext(ctx).Fatalf("failed to running migration: %+v", err)
		os.Exit(1)
	}

	return db
}
