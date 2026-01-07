package fiber_inbound_adapter_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"

	fiber_inbound_adapter "prabogo/internal/adapter/inbound/fiber"
	"prabogo/internal/domain"
	"prabogo/internal/model"
	mock_outbound_port "prabogo/tests/mocks/port"
)

func TestMiddlewareAdapter(t *testing.T) {
	Convey("Test Middleware Adapter", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockDatabasePort := mock_outbound_port.NewMockDatabasePort(mockCtrl)
		mockCachePort := mock_outbound_port.NewMockCachePort(mockCtrl)
		mockMessagePort := mock_outbound_port.NewMockMessagePort(mockCtrl)
		mockClientDatabasePort := mock_outbound_port.NewMockClientDatabasePort(mockCtrl)
		mockClientCachePort := mock_outbound_port.NewMockClientCachePort(mockCtrl)

		mockDatabasePort.EXPECT().Client().Return(mockClientDatabasePort).AnyTimes()
		mockCachePort.EXPECT().Client().Return(mockClientCachePort).AnyTimes()
		mockMessagePort.EXPECT().Client().Return(mock_outbound_port.NewMockClientMessagePort(mockCtrl)).AnyTimes()

		dom := domain.NewDomain(mockDatabasePort, mockMessagePort, mockCachePort)
		adapter := fiber_inbound_adapter.NewAdapter(dom)

		Convey("InternalAuth", func() {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				return adapter.Middleware().InternalAuth(c)
			})
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			Convey("Missing Authorization header", func() {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})

			Convey("Empty bearer token", func() {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer ")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})

			Convey("Invalid bearer token", func() {
				os.Setenv("INTERNAL_KEY", "valid-key")
				defer os.Unsetenv("INTERNAL_KEY")

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer invalid-key")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})

			Convey("Valid bearer token", func() {
				os.Setenv("INTERNAL_KEY", "valid-key")
				defer os.Unsetenv("INTERNAL_KEY")

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer valid-key")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Malformed authorization header", func() {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Basic abc123")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})
		})

		Convey("ClientAuth", func() {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				return adapter.Middleware().ClientAuth(c)
			})
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			clientOutput := model.Client{
				ID: 1,
				ClientInput: model.ClientInput{
					Name:      "Test Client",
					BearerKey: "valid-client-key",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}

			Convey("Missing Authorization header", func() {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})

			Convey("Client exists in cache", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(clientOutput, nil).Times(1)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer valid-client-key")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Client exists in database (cache miss)", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(true, nil).Times(1)
				mockClientDatabasePort.EXPECT().FindByFilter(gomock.Any(), gomock.Any()).Return([]model.Client{clientOutput}, nil).Times(1)
				mockClientCachePort.EXPECT().Set(gomock.Any()).Return(nil).Times(1)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer valid-client-key")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)
			})

			Convey("Client does not exist", func() {
				mockClientCachePort.EXPECT().Get(gomock.Any()).Return(model.Client{}, redis.Nil).Times(1)
				mockClientDatabasePort.EXPECT().IsExists(gomock.Any()).Return(false, nil).Times(1)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer nonexistent-key")
				resp, err := app.Test(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})
		})
	})
}
