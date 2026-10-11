package fiber_inbound_adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	. "github.com/smartystreets/goconvey/convey"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestObservabilityMiddlewareRouteLabel(t *testing.T) {
	Convey("Observability middleware records real route on auth failure", t, func() {
		reader := metric.NewManualReader()
		mp := metric.NewMeterProvider(
			metric.WithReader(reader),
			metric.WithResource(resource.Default()),
		)
		otel.SetMeterProvider(mp)
		defer func() { otel.SetMeterProvider(otel.GetMeterProvider()) }()

		app := fiber.New()
		app.Use(observabilityMiddleware())
		client := app.Group("/v1")
		client.Use(func(c *fiber.Ctx) error {
			// Simulate the client auth middleware: reject requests without a
			// valid bearer token, short-circuiting before the handler.
			if c.Get("Authorization") == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
			}
			return c.Next()
		})
		client.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("OK") })

		req := httptest.NewRequest(http.MethodGet, "/v1/ping", nil)
		resp, err := app.Test(req)
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusUnauthorized)

		assertRecordedRoute(t, reader, "/v1/ping")
	})
}

// assertRecordedRoute collects the request counter and asserts that a data
// point carrying the given http.route label exists.
func assertRecordedRoute(t *testing.T, reader *metric.ManualReader, route string) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	So(err, ShouldBeNil)

	var routeAttr attribute.KeyValue
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "prabogo.http.server.request.count" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			So(ok, ShouldBeTrue)
			for _, dp := range sum.DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					if kv.Key == "http.route" {
						routeAttr = kv
						found = true
					}
				}
			}
		}
	}
	So(found, ShouldBeTrue)
	So(routeAttr.Value.AsString(), ShouldEqual, route)
}
