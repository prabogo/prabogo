package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	meterName = "prabogo.metrics"
)

var (
	httpOnce        sync.Once
	messageOnce     sync.Once
	httpServer      *httpMetrics
	messageConsumer *messageMetrics
)

type httpMetrics struct {
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
}

type messageMetrics struct {
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
}

func GetHTTPServer() *httpMetrics {
	httpOnce.Do(func() {
		meter := otel.Meter(meterName)

		counter, err := meter.Int64Counter(
			"prabogo.http.server.request.count",
			metric.WithDescription("Total number of HTTP requests handled by the server"),
			metric.WithUnit("{request}"),
		)
		if err != nil {
			counter = nil
		}

		histogram, err := meter.Float64Histogram(
			"prabogo.http.server.request.duration",
			metric.WithDescription("HTTP server request duration in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			histogram = nil
		}

		httpServer = &httpMetrics{
			requestCounter: counter,
			requestLatency: histogram,
		}
	})
	return httpServer
}

func (m *httpMetrics) RecordHTTPRequest(ctx context.Context, method, route string, statusCode int, duration float64) {
	if m == nil {
		return
	}

	isError := statusCode >= 400
	attrs := []attribute.KeyValue{
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(route),
		semconv.HTTPResponseStatusCodeKey.Int(statusCode),
		attribute.Bool("error", isError),
	}

	// Do NOT add trace_id as a metric label — it creates a separate time series
	// per request (high cardinality), which breaks rate() and
	// histogram_quantile() queries in Mimir. The OTel SDK natively attaches
	// exemplars (trace_id + span_id) to metric data points when a sampled span
	// is in the context; the collector's prometheusremotewrite exporter forwards
	// them to Mimir, and Grafana's exemplarTraceIdLabel picks them up.

	if m.requestCounter != nil {
		m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if m.requestLatency != nil {
		m.requestLatency.Record(ctx, duration, metric.WithAttributes(attrs...))
	}
}

func GetMessageConsumer() *messageMetrics {
	messageOnce.Do(func() {
		meter := otel.Meter(meterName)

		counter, err := meter.Int64Counter(
			"prabogo.message.consumer.request.count",
			metric.WithDescription("Total number of messages consumed from the message broker"),
			metric.WithUnit("{message}"),
		)
		if err != nil {
			counter = nil
		}

		histogram, err := meter.Float64Histogram(
			"prabogo.message.consumer.request.duration",
			metric.WithDescription("Message consumer processing duration in seconds"),
			metric.WithUnit("s"),
		)
		if err != nil {
			histogram = nil
		}

		messageConsumer = &messageMetrics{
			requestCounter: counter,
			requestLatency: histogram,
		}
	})
	return messageConsumer
}

func (m *messageMetrics) RecordMessage(ctx context.Context, record MessageRecord) {
	if m == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String("rabbitmq"),
		semconv.MessagingDestinationNameKey.String(record.Exchange),
		attribute.String("messaging.destination.queue", record.Queue),
		attribute.String("messaging.operation", record.Operation),
		attribute.Bool("error", record.Error),
	}

	// Do NOT add trace_id as a metric label — see RecordHTTPRequest comment.

	if m.requestCounter != nil {
		m.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	if m.requestLatency != nil {
		m.requestLatency.Record(ctx, record.Duration, metric.WithAttributes(attrs...))
	}
}

type MessageRecord struct {
	Exchange  string
	Queue     string
	Operation string
	Error     bool
	Duration  float64
}

func RecordMessage(ctx context.Context, record MessageRecord) {
	GetMessageConsumer().RecordMessage(ctx, record)
}
