package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"

	"prabogo/internal/domain"
	"prabogo/internal/model"
	mock_outbound_port "prabogo/tests/mocks/port"
)

func TestClient(t *testing.T) {
	Convey("Test Client", t, func() {
		mockCtrl := gomock.NewController(t)

		defer mockCtrl.Finish()

		mockDatabasePort := mock_outbound_port.NewMockDatabasePort(mockCtrl)
		mockCachePort := mock_outbound_port.NewMockCachePort(mockCtrl)
		mockMessagePort := mock_outbound_port.NewMockMessagePort(mockCtrl)

		mockClientDatabasePort := mock_outbound_port.NewMockClientDatabasePort(mockCtrl)
		mockClientMessagePort := mock_outbound_port.NewMockClientMessagePort(mockCtrl)
		mockClientCachePort := mock_outbound_port.NewMockClientCachePort(mockCtrl)

		mockDatabasePort.EXPECT().Client().Return(mockClientDatabasePort).AnyTimes()
		mockMessagePort.EXPECT().Client().Return(mockClientMessagePort).AnyTimes()
		mockCachePort.EXPECT().Client().Return(mockClientCachePort).AnyTimes()

		clientDomain := domain.NewDomain(mockDatabasePort, mockMessagePort, mockCachePort)

		inputs := []model.ClientInput{
			model.ClientInput{
				Name: "Test Client",
			},
		}

		outputs := []model.Client{
			model.Client{
				ID: 1,
				ClientInput: model.ClientInput{
					Name:      "Test Client",
					BearerKey: "test-bearer-key",
					UpdatedAt: time.Now(),
					CreatedAt: time.Now(),
				},
			},
		}

		filter := model.ClientFilter{
			BearerKeys: []string{"test-bearer-key"},
			IDs:        []int{1},
			Names:      []string{"Test Client"},
		}

		Convey("Upsert", func() {
			Convey("Input is empty", func() {
				_, err := clientDomain.Client().Upsert(context.Background(), []model.ClientInput{})
				So(err, ShouldNotBeNil)
			})

			Convey("Database client upsert error", func() {
				mockClientDatabasePort.EXPECT().Upsert(gomock.Any()).Return(errors.New("error")).Times(1)

				_, err := clientDomain.Client().Upsert(context.Background(), inputs)
				So(err, ShouldNotBeNil)
			})

			Convey("Database client find by filter error", func() {
				mockClientDatabasePort.EXPECT().Upsert(gomock.Any()).Return(nil).Times(1)
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).Times(1)

				_, err := clientDomain.Client().Upsert(context.Background(), inputs)
				So(err, ShouldNotBeNil)
			})

			Convey("Success", func() {
				mockClientDatabasePort.EXPECT().Upsert(gomock.Any()).Return(nil).Times(1)
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(outputs, nil).Times(1)

				results, err := clientDomain.Client().Upsert(context.Background(), inputs)
				So(err, ShouldBeNil)
				So(results, ShouldNotBeEmpty)
				So(results[0].Name, ShouldEqual, "Test Client")
			})
		})

		Convey("FindByFilter", func() {
			Convey("Filter is empty", func() {
				_, err := clientDomain.Client().FindByFilter(context.Background(), model.ClientFilter{})
				So(err, ShouldNotBeNil)
			})

			Convey("Database client find by filter error", func() {
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).Times(1)

				_, err := clientDomain.Client().FindByFilter(context.Background(), filter)
				So(err, ShouldNotBeNil)
			})

			Convey("Success", func() {
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(outputs, nil).Times(1)

				results, err := clientDomain.Client().FindByFilter(context.Background(), filter)
				So(err, ShouldBeNil)
				So(results, ShouldNotBeEmpty)
				So(results[0].Name, ShouldEqual, "Test Client")
			})
		})

		Convey("DeleteByFilter", func() {
			Convey("Filter is empty", func() {
				err := clientDomain.Client().DeleteByFilter(context.Background(), model.ClientFilter{})
				So(err, ShouldNotBeNil)
			})

			Convey("Database client delete by filter error", func() {
				mockClientDatabasePort.EXPECT().DeleteByFilter(gomock.Any()).Return(errors.New("error")).Times(1)

				err := clientDomain.Client().DeleteByFilter(context.Background(), filter)
				So(err, ShouldNotBeNil)
			})

			Convey("Success", func() {
				mockClientDatabasePort.EXPECT().DeleteByFilter(gomock.Any()).Return(nil).Times(1)

				err := clientDomain.Client().DeleteByFilter(context.Background(), filter)
				So(err, ShouldBeNil)
			})
		})

		Convey("PublishUpsert", func() {
			Convey("Input is empty", func() {
				err := clientDomain.Client().PublishUpsert(context.Background(), []model.ClientInput{})
				So(err, ShouldNotBeNil)
			})

			Convey("Message client publish upsert error", func() {
				mockClientMessagePort.EXPECT().PublishUpsert(gomock.Any()).Return(errors.New("error")).Times(1)

				err := clientDomain.Client().PublishUpsert(context.Background(), inputs)
				So(err, ShouldNotBeNil)
			})

			Convey("Success", func() {
				mockClientMessagePort.EXPECT().PublishUpsert(gomock.Any()).Return(nil).Times(1)

				err := clientDomain.Client().PublishUpsert(context.Background(), inputs)
				So(err, ShouldBeNil)
			})
		})

		Convey("IsExists", func() {
			Convey("Bearer key is empty", func() {
				_, err := clientDomain.Client().IsExists(context.Background(), "")
				So(err, ShouldNotBeNil)
			})

			Convey("Cache client get error", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, errors.New("error")).Times(1)

				_, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldNotBeNil)
			})

			Convey("Database client is exists error", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(false, errors.New("error")).Times(1)

				_, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldNotBeNil)
			})

			Convey("Database client find by filter error", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(true, nil).Times(1)

				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(nil, errors.New("error")).Times(1)

				_, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldNotBeNil)
			})

			Convey("Cache client set error", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(true, nil).Times(1)

				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(outputs, nil).Times(1)
				mockClientCachePort.EXPECT().Set(gomock.Any()).Return(errors.New("error")).Times(1)

				_, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldNotBeNil)
			})

			Convey("Success", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(true, nil).Times(1)
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return(outputs, nil).Times(1)
				mockClientCachePort.EXPECT().Set(gomock.Any()).Return(nil).Times(1)

				result, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldBeNil)
				So(result, ShouldBeTrue)
			})

			Convey("Cache client exists", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(outputs[0], nil).Times(1)

				_, err := clientDomain.Client().IsExists(context.Background(), "test-bearer-key")
				So(err, ShouldBeNil)
			})
		})
	})
}
