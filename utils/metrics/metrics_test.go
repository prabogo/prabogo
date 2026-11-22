package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"prabogo/utils/activity"
)

// TestMetrics is the single entry point for all metrics tests. Because the
// metrics package uses a package-level sync.Once to initialize the meter
// (bound to the first meter provider set), all sub-tests must share a single
// meter provider. This test sets it up once and runs sub-tests via t.Run.
func TestMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(resource.Default()),
	)
	otel.SetMeterProvider(mp)
	defer mp.Shutdown(context.Background())

	// Force initialization of both metric sets against the test provider.
	_ = GetHTTPServer()
	_ = GetMessageConsumer()

	// Helper to collect and find a counter.
	collect := func() metricdata.ResourceMetrics {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatalf("reader.Collect: %v", err)
		}
		return rm
	}
	findCounter := func(rm metricdata.ResourceMetrics, name string) *metricdata.Sum[int64] {
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != name {
					continue
				}
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Fatalf("metric %q is not a Sum[int64], got %T", name, m.Data)
				}
				return &sum
			}
		}
		t.Fatalf("counter %q not found in metrics", name)
		return nil
	}
	findHistogram := func(rm metricdata.ResourceMetrics, name string) *metricdata.Histogram[float64] {
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != name {
					continue
				}
				hist, ok := m.Data.(metricdata.Histogram[float64])
				if !ok {
					t.Fatalf("metric %q is not a Histogram[float64], got %T", name, m.Data)
				}
				return &hist
			}
		}
		t.Fatalf("histogram %q not found in metrics", name)
		return nil
	}
	attrValue := func(attrs []attribute.KeyValue, key string) (attribute.Value, bool) {
		for _, kv := range attrs {
			if string(kv.Key) == key {
				return kv.Value, true
			}
		}
		return attribute.Value{}, false
	}

	t.Run("HTTPRequestSuccess", func(t *testing.T) {
		m := GetHTTPServer()
		m.RecordHTTPRequest(context.Background(), "GET", "/v1/ping", 200, 0.05)

		rm := collect()
		counter := findCounter(rm, "prabogo.http.server.request.count")
		if len(counter.DataPoints) != 1 {
			t.Fatalf("expected 1 data point, got %d", len(counter.DataPoints))
		}
		dp := counter.DataPoints[0]
		if dp.Value != 1 {
			t.Errorf("expected counter value 1, got %d", dp.Value)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "http.request.method"); !ok || v.AsString() != "GET" {
			t.Errorf("expected http.request.method=GET, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "http.route"); !ok || v.AsString() != "/v1/ping" {
			t.Errorf("expected http.route=/v1/ping, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "http.response.status_code"); !ok || v.AsInt64() != 200 {
			t.Errorf("expected http.response.status_code=200, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "error"); !ok || v.AsBool() != false {
			t.Errorf("expected error=false, got %v", v)
		}
	})

	t.Run("HTTPRequestError", func(t *testing.T) {
		m := GetHTTPServer()
		m.RecordHTTPRequest(context.Background(), "POST", "/v1/upsert", 500, 0.1)

		rm := collect()
		counter := findCounter(rm, "prabogo.http.server.request.count")
		// Find the data point with status 500.
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "http.response.status_code"); ok && v.AsInt64() == 500 {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point with status 500, got %d points", len(counter.DataPoints))
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "error"); !ok || v.AsBool() != true {
			t.Errorf("expected error=true for status 500, got %v", v)
		}
	})

	t.Run("HTTPRequestBoundaryStatus", func(t *testing.T) {
		m := GetHTTPServer()
		// 399 should NOT be an error; 400 should be.
		m.RecordHTTPRequest(context.Background(), "GET", "/boundary399", 399, 0.01)
		m.RecordHTTPRequest(context.Background(), "GET", "/boundary400", 400, 0.01)

		rm := collect()
		counter := findCounter(rm, "prabogo.http.server.request.count")
		// Find the two data points we just recorded (by route).
		var found399, found400 bool
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "http.route"); ok {
				switch v.AsString() {
				case "/boundary399":
					found399 = true
					if ev, ok := attrValue(p.Attributes.ToSlice(), "error"); !ok || ev.AsBool() != false {
						t.Errorf("expected error=false for status 399, got %v", ev)
					}
				case "/boundary400":
					found400 = true
					if ev, ok := attrValue(p.Attributes.ToSlice(), "error"); !ok || ev.AsBool() != true {
						t.Errorf("expected error=true for status 400, got %v", ev)
					}
				}
			}
		}
		if !found399 {
			t.Error("expected a data point for /boundary399")
		}
		if !found400 {
			t.Error("expected a data point for /boundary400")
		}
	})

	t.Run("HTTPRequestHistogram", func(t *testing.T) {
		m := GetHTTPServer()
		m.RecordHTTPRequest(context.Background(), "GET", "/v1/hist", 200, 0.123)

		rm := collect()
		hist := findHistogram(rm, "prabogo.http.server.request.duration")
		// Find the data point for /v1/hist.
		var dp metricdata.HistogramDataPoint[float64]
		found := false
		for _, p := range hist.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "http.route"); ok && v.AsString() == "/v1/hist" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a histogram data point for /v1/hist, got %d points", len(hist.DataPoints))
		}
		if dp.Sum != 0.123 {
			t.Errorf("expected histogram sum 0.123, got %f", dp.Sum)
		}
		if dp.Count != 1 {
			t.Errorf("expected histogram count 1, got %d", dp.Count)
		}
	})

	t.Run("HTTPRequestWithTraceID", func(t *testing.T) {
		tid, _ := trace.TraceIDFromHex("0abc0123456789abcdef0123456789a1")
		sid, _ := trace.SpanIDFromHex("0def0123456789a1")
		sctx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    tid,
			SpanID:     sid,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sctx)

		m := GetHTTPServer()
		m.RecordHTTPRequest(ctx, "GET", "/v1/trace", 200, 0.05)

		rm := collect()
		counter := findCounter(rm, "prabogo.http.server.request.count")
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "http.route"); ok && v.AsString() == "/v1/trace" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point for /v1/trace")
		}
		// trace_id should NOT be a metric label — it causes high cardinality.
		// The OTel SDK attaches exemplars natively via the span context.
		if v, ok := attrValue(dp.Attributes.ToSlice(), "trace_id"); ok {
			t.Errorf("expected NO trace_id label on metric (high cardinality), got %q", v.AsString())
		}
	})

	t.Run("HTTPRequestNoTraceID", func(t *testing.T) {
		m := GetHTTPServer()
		m.RecordHTTPRequest(context.Background(), "GET", "/v1/notrace", 200, 0.05)

		rm := collect()
		counter := findCounter(rm, "prabogo.http.server.request.count")
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "http.route"); ok && v.AsString() == "/v1/notrace" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point for /v1/notrace")
		}
		if _, ok := attrValue(dp.Attributes.ToSlice(), "trace_id"); ok {
			t.Error("expected NO trace_id attribute when ctx has no span")
		}
	})

	t.Run("MessageSuccess", func(t *testing.T) {
		m := GetMessageConsumer()
		m.RecordMessage(context.Background(), MessageRecord{
			Exchange:  "upsert_client",
			Queue:     "upsert_client.queue",
			Operation: "receive",
			Error:     false,
			Duration:  0.05,
		})

		rm := collect()
		counter := findCounter(rm, "prabogo.message.consumer.request.count")
		if len(counter.DataPoints) != 1 {
			t.Fatalf("expected 1 data point, got %d", len(counter.DataPoints))
		}
		dp := counter.DataPoints[0]
		if dp.Value != 1 {
			t.Errorf("expected counter value 1, got %d", dp.Value)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "messaging.system"); !ok || v.AsString() != "rabbitmq" {
			t.Errorf("expected messaging.system=rabbitmq, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "messaging.destination.name"); !ok || v.AsString() != "upsert_client" {
			t.Errorf("expected messaging.destination.name=upsert_client, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "messaging.destination.queue"); !ok || v.AsString() != "upsert_client.queue" {
			t.Errorf("expected messaging.destination.queue, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "messaging.operation"); !ok || v.AsString() != "receive" {
			t.Errorf("expected messaging.operation=receive, got %v", v)
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "error"); !ok || v.AsBool() != false {
			t.Errorf("expected error=false, got %v", v)
		}
	})

	t.Run("MessageError", func(t *testing.T) {
		m := GetMessageConsumer()
		m.RecordMessage(context.Background(), MessageRecord{
			Exchange:  "upsert_client_err",
			Queue:     "upsert_client.queue",
			Operation: "receive",
			Error:     true,
			Duration:  0.05,
		})

		rm := collect()
		counter := findCounter(rm, "prabogo.message.consumer.request.count")
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "messaging.destination.name"); ok && v.AsString() == "upsert_client_err" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point for upsert_client_err")
		}
		if v, ok := attrValue(dp.Attributes.ToSlice(), "error"); !ok || v.AsBool() != true {
			t.Errorf("expected error=true, got %v", v)
		}
	})

	t.Run("MessageHistogram", func(t *testing.T) {
		m := GetMessageConsumer()
		m.RecordMessage(context.Background(), MessageRecord{
			Exchange:  "upsert_client_hist",
			Queue:     "upsert_client.queue",
			Operation: "receive",
			Error:     false,
			Duration:  0.25,
		})

		rm := collect()
		hist := findHistogram(rm, "prabogo.message.consumer.request.duration")
		var dp metricdata.HistogramDataPoint[float64]
		found := false
		for _, p := range hist.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "messaging.destination.name"); ok && v.AsString() == "upsert_client_hist" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a histogram data point for upsert_client_hist")
		}
		if dp.Sum != 0.25 {
			t.Errorf("expected histogram sum 0.25, got %f", dp.Sum)
		}
	})

	t.Run("MessageWithTraceID", func(t *testing.T) {
		tid, _ := trace.TraceIDFromHex("0abc0123456789abcdef0123456789a1")
		sid, _ := trace.SpanIDFromHex("0def0123456789a1")
		sctx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    tid,
			SpanID:     sid,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sctx)

		m := GetMessageConsumer()
		m.RecordMessage(ctx, MessageRecord{
			Exchange:  "upsert_client_trace",
			Queue:     "upsert_client.queue",
			Operation: "receive",
			Error:     false,
			Duration:  0.05,
		})

		rm := collect()
		counter := findCounter(rm, "prabogo.message.consumer.request.count")
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "messaging.destination.name"); ok && v.AsString() == "upsert_client_trace" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point for upsert_client_trace")
		}
		// trace_id should NOT be a metric label — it causes high cardinality.
		if v, ok := attrValue(dp.Attributes.ToSlice(), "trace_id"); ok {
			t.Errorf("expected NO trace_id label on message metric (high cardinality), got %q", v.AsString())
		}
	})

	t.Run("MessageFromActivityContext", func(t *testing.T) {
		tid, _ := trace.TraceIDFromHex("1abc0123456789abcdef0123456789a1")
		sid, _ := trace.SpanIDFromHex("1def0123456789a1")
		sctx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    tid,
			SpanID:     sid,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), sctx)
		ctx = activity.NewContext("message_client_upsert", ctx)

		m := GetMessageConsumer()
		m.RecordMessage(ctx, MessageRecord{
			Exchange:  "upsert_client_activity",
			Queue:     "upsert_client.queue",
			Operation: "receive",
			Error:     false,
			Duration:  0.05,
		})

		rm := collect()
		counter := findCounter(rm, "prabogo.message.consumer.request.count")
		var dp metricdata.DataPoint[int64]
		found := false
		for _, p := range counter.DataPoints {
			if v, ok := attrValue(p.Attributes.ToSlice(), "messaging.destination.name"); ok && v.AsString() == "upsert_client_activity" {
				dp = p
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a data point for upsert_client_activity")
		}
		// trace_id should NOT be a metric label — it causes high cardinality.
		if v, ok := attrValue(dp.Attributes.ToSlice(), "trace_id"); ok {
			t.Errorf("expected NO trace_id label on message metric (high cardinality), got %q", v.AsString())
		}
	})
}

// TestNilReceivers ensures calling record methods on nil receivers does not
// panic. These run in a separate test (no meter provider needed).
func TestNilReceivers(t *testing.T) {
	var hm *httpMetrics
	hm.RecordHTTPRequest(context.Background(), "GET", "/", 200, 0.01)

	var mm *messageMetrics
	mm.RecordMessage(context.Background(), MessageRecord{})
}
