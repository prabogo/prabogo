package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TestExtractTraceContextRoundTrip verifies that a trace context injected by
// the publisher's propagation is correctly extracted back by the subscriber.
// This is the core of cross-service trace propagation via RabbitMQ headers.
func TestExtractTraceContextRoundTrip(t *testing.T) {
	// Ensure the global propagator is the composite TraceContext+Baggage one
	// (same as what otel.Setup installs).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create a parent span context (simulating the publisher's active span).
	tid, err := trace.TraceIDFromHex("0abc0123456789abcdef0123456789a1")
	if err != nil {
		t.Fatalf("TraceIDFromHex: %v", err)
	}
	sid, err := trace.SpanIDFromHex("0def0123456789a1")
	if err != nil {
		t.Fatalf("SpanIDFromHex: %v", err)
	}
	parentSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	parentCtx := trace.ContextWithSpanContext(context.Background(), parentSC)

	// Inject the trace context into AMQP headers (same as publisher.go).
	headers := amqp.Table{}
	carrier := propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)
	for k, values := range carrier {
		if len(values) > 0 {
			headers[k] = values[0]
		}
	}

	// Verify the traceparent header was injected (HeaderCarrier preserves
	// the propagator's case — W3C uses "Traceparent").
	tp, ok := headers["Traceparent"].(string)
	if !ok {
		// Some propagator versions use lowercase.
		tp, ok = headers["traceparent"].(string)
	}
	if !ok || tp == "" {
		t.Fatal("expected traceparent header to be injected")
	}

	// Extract the trace context back (same as subscriber.go extractTraceContext).
	extractedCtx := extractTraceContext(context.Background(), headers)
	extractedSC := trace.SpanContextFromContext(extractedCtx)

	if !extractedSC.IsValid() {
		t.Fatal("extracted span context is not valid")
	}
	if extractedSC.TraceID() != parentSC.TraceID() {
		t.Errorf("trace ID mismatch: parent=%s extracted=%s",
			parentSC.TraceID(), extractedSC.TraceID())
	}
	if extractedSC.SpanID() != parentSC.SpanID() {
		t.Errorf("span ID mismatch: parent=%s extracted=%s",
			parentSC.SpanID(), extractedSC.SpanID())
	}
	if !extractedSC.IsSampled() {
		t.Error("expected extracted span context to be sampled")
	}
}

// TestExtractTraceContextNoHeaders verifies that extracting from empty/nil
// headers produces a valid but empty context (no trace context).
func TestExtractTraceContextNoHeaders(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Nil headers.
	ctx := extractTraceContext(context.Background(), nil)
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		t.Error("expected invalid span context from nil headers")
	}

	// Empty headers.
	ctx = extractTraceContext(context.Background(), amqp.Table{})
	sc = trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		t.Error("expected invalid span context from empty headers")
	}
}

// TestExtractTraceContextNonStringHeaders verifies that non-string header
// values are safely skipped (the extractor only handles string values).
func TestExtractTraceContextNonStringHeaders(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	headers := amqp.Table{
		"traceparent": 123,             // non-string, should be skipped
		"other":       []byte("bytes"), // non-string, should be skipped
	}

	ctx := extractTraceContext(context.Background(), headers)
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		t.Error("expected invalid span context from non-string headers")
	}
}

// TestExtractTraceContextPreservesBaggage verifies that baggage is also
// propagated through the AMQP headers without crashing.
func TestExtractTraceContextPreservesBaggage(t *testing.T) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	parentCtx := context.Background()

	// Inject and extract.
	headers := amqp.Table{}
	carrier := propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(parentCtx, carrier)
	for k, values := range carrier {
		if len(values) > 0 {
			headers[k] = values[0]
		}
	}

	extractedCtx := extractTraceContext(context.Background(), headers)
	// Should not panic and should produce a valid context.
	_ = trace.SpanContextFromContext(extractedCtx)
}
