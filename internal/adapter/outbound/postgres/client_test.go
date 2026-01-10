package postgres_outbound_adapter_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/smartystreets/goconvey/convey"

	postgres_outbound_adapter "prabogo/internal/adapter/outbound/postgres"
	"prabogo/internal/model"
)

func TestClientAdapter(t *testing.T) {
	Convey("Test Postgres Client Adapter", t, func() {
		db, mock, err := sqlmock.New()
		So(err, ShouldBeNil)
		defer db.Close()

		adapter := postgres_outbound_adapter.NewClientAdapter(db)

		now := time.Now()
		inputs := []model.ClientInput{
			{
				Name:      "Test Client",
				BearerKey: "test-key",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		filter := model.ClientFilter{
			IDs: []int{1},
		}

		Convey("Upsert", func() {
			Convey("Success", func() {
				mock.ExpectExec("INSERT INTO \"clients\"").
					WillReturnResult(sqlmock.NewResult(1, 1))

				err := adapter.Upsert(inputs)
				So(err, ShouldBeNil)
				So(mock.ExpectationsWereMet(), ShouldBeNil)
			})

			Convey("Database error", func() {
				mock.ExpectExec("INSERT INTO \"clients\"").
					WillReturnError(sqlmock.ErrCancelled)

				err := adapter.Upsert(inputs)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("FindByFilter", func() {
			Convey("Success", func() {
				rows := sqlmock.NewRows([]string{"id", "name", "bearer_key", "created_at", "updated_at"}).
					AddRow(1, "Test Client", "test-key", now, now)

				mock.ExpectQuery("SELECT \\* FROM \"clients\"").
					WillReturnRows(rows)

				results, err := adapter.FindByFilter(filter, false)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(results[0].Name, ShouldEqual, "Test Client")
				So(mock.ExpectationsWereMet(), ShouldBeNil)
			})

			Convey("With lock", func() {
				rows := sqlmock.NewRows([]string{"id", "name", "bearer_key", "created_at", "updated_at"}).
					AddRow(1, "Test Client", "test-key", now, now)

				mock.ExpectQuery("SELECT \\* FROM \"clients\"").
					WillReturnRows(rows)

				results, err := adapter.FindByFilter(filter, true)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 1)
				So(mock.ExpectationsWereMet(), ShouldBeNil)
			})

			Convey("Query error", func() {
				mock.ExpectQuery("SELECT \\* FROM \"clients\"").
					WillReturnError(sqlmock.ErrCancelled)

				_, err := adapter.FindByFilter(filter, false)
				So(err, ShouldNotBeNil)
			})

			Convey("Empty result", func() {
				rows := sqlmock.NewRows([]string{"id", "name", "bearer_key", "created_at", "updated_at"})

				mock.ExpectQuery("SELECT \\* FROM \"clients\"").
					WillReturnRows(rows)

				results, err := adapter.FindByFilter(filter, false)
				So(err, ShouldBeNil)
				So(len(results), ShouldEqual, 0)
			})
		})

		Convey("DeleteByFilter", func() {
			Convey("Success", func() {
				rows := sqlmock.NewRows([]string{})
				mock.ExpectQuery("DELETE FROM \"clients\"").
					WillReturnRows(rows)

				err := adapter.DeleteByFilter(filter)
				So(err, ShouldBeNil)
				So(mock.ExpectationsWereMet(), ShouldBeNil)
			})

			Convey("Query error", func() {
				mock.ExpectQuery("DELETE FROM \"clients\"").
					WillReturnError(sqlmock.ErrCancelled)

				err := adapter.DeleteByFilter(filter)
				So(err, ShouldNotBeNil)
			})
		})

		Convey("IsExists", func() {
			Convey("Exists", func() {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery("SELECT \"id\" FROM \"clients\"").
					WillReturnRows(rows)

				exists, err := adapter.IsExists("test-key")
				So(err, ShouldBeNil)
				So(exists, ShouldBeTrue)
				So(mock.ExpectationsWereMet(), ShouldBeNil)
			})

			Convey("Not exists", func() {
				rows := sqlmock.NewRows([]string{"id"})
				mock.ExpectQuery("SELECT \"id\" FROM \"clients\"").
					WillReturnRows(rows)

				exists, err := adapter.IsExists("nonexistent")
				So(err, ShouldBeNil)
				So(exists, ShouldBeFalse)
			})

			Convey("Query error", func() {
				mock.ExpectQuery("SELECT \"id\" FROM \"clients\"").
					WillReturnError(sqlmock.ErrCancelled)

				_, err := adapter.IsExists("test-key")
				So(err, ShouldNotBeNil)
			})
		})
	})
}
