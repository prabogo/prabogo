package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"prabogo/utils/activity"
)

// newTestTracerProvider installs a tracer provider backed by an in-memory
// span recorder so tests can assert on the spans produced by the helpers.
func newTestTracerProvider() (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(rec),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return rec, tp
}

// endedStubs returns the SpanStubs from the recorder's ended spans.
func endedStubs(rec *tracetest.SpanRecorder) tracetest.SpanStubs {
	return tracetest.SpanStubsFromReadOnlySpans(rec.Ended())
}

func TestStartSpanCreatesSpan(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test.tracer", "test.span")
	defer span.End()

	if !span.IsRecording() {
		t.Fatal("expected span to be recording")
	}
	if !span.SpanContext().HasTraceID() {
		t.Fatal("expected span to have a trace ID")
	}

	// The span must be registered as the active span in the returned ctx.
	active := trace.SpanFromContext(ctx)
	if active.SpanContext().SpanID() != span.SpanContext().SpanID() {
		t.Fatal("returned ctx does not carry the created span")
	}

	// Ending the span should flush it to the recorder.
	span.End()
	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
}

func TestStartSpanCarriesActivityAttributes(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	ctx := activity.NewContext("my_action")
	ctx = activity.WithClientID(ctx, "client-123")

	_, span := StartSpan(ctx, "test.tracer", "test.span")
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)

	if v, ok := attrs.Value("activity.action"); !ok || v.AsString() != "my_action" {
		t.Errorf("expected activity.action=my_action, got %v", v)
	}
	if v, ok := attrs.Value("activity.client_id"); !ok || v.AsString() != "client-123" {
		t.Errorf("expected activity.client_id=client-123, got %v", v)
	}
	// transaction_id is always set by activity.NewContext.
	if v, ok := attrs.Value("activity.transaction_id"); !ok || v.AsString() == "" {
		t.Errorf("expected non-empty activity.transaction_id, got %v", v)
	}
}

func TestStartSpanChildOfParent(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	tracer := otel.Tracer("parent")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")

	_, childSpan := StartSpan(parentCtx, "test.tracer", "child")
	childSpan.End()
	parentSpan.End()

	spans := endedStubs(rec)
	if len(spans) != 2 {
		t.Fatalf("expected 2 exported spans, got %d", len(spans))
	}

	// The child must share the parent's trace ID.
	if parentSpan.SpanContext().TraceID() != childSpan.SpanContext().TraceID() {
		t.Fatal("child span does not share parent trace ID")
	}
	// The child's parent must be the parent span (assert via the exported
	// SpanStub.Parent field).
	var childStub tracetest.SpanStub
	for _, s := range endedStubs(rec) {
		if s.Name == "child" {
			childStub = s
		}
	}
	if childStub.Parent.SpanID() != parentSpan.SpanContext().SpanID() {
		t.Fatal("child span parent ID does not match parent span ID")
	}
}

func TestStartSpanDBSemconv(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "db.span", WithDBSemconv())
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)
	if v, ok := attrs.Value("db.system"); !ok {
		t.Errorf("expected db.system attribute, got %v", v)
	}
}

func TestStartSpanMQSemconvPublish(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "mq.publish", WithMQSemconv(false))
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)
	if v, ok := attrs.Value("messaging.system"); !ok {
		t.Errorf("expected messaging.system attribute, got %v", v)
	}
	if v, ok := attrs.Value("messaging.operation.type"); !ok || v.AsString() != "publish" {
		t.Errorf("expected messaging.operation.type=publish, got %v", v)
	}
}

func TestStartSpanMQSemconvConsume(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "mq.consume", WithMQSemconv(true))
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)
	if v, ok := attrs.Value("messaging.operation.type"); !ok || v.AsString() != "receive" {
		t.Fatalf("expected messaging.operation.type=receive, got %v", v)
	}
}

func TestRecordError(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span")
	err := errors.New("boom")
	RecordError(span, err)
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("expected span status Error, got %v", spans[0].Status.Code)
	}
}

func TestRecordErrorNil(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span")
	RecordError(span, nil)
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	if spans[0].Status.Code == codes.Error {
		t.Error("nil error should not set span status to Error")
	}
}

func TestWithSpanAttributeKey(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		want  attribute.Value
	}{
		{"str", "hello", attribute.StringValue("hello")},
		{"int", 42, attribute.Int64Value(42)},
		{"bool", true, attribute.BoolValue(true)},
		{"float", 3.14, attribute.Float64Value(3.14)},
	}
	for _, tt := range tests {
		kv := WithSpanAttributeKey(tt.key, tt.value)
		if kv.Key != attribute.Key(tt.key) {
			t.Errorf("expected key %q, got %q", tt.key, kv.Key)
		}
		if kv.Value.Type() != tt.want.Type() {
			t.Errorf("expected type %v, got %v", tt.want.Type(), kv.Value.Type())
		}
	}
}

func TestWithSpanAttributeKeyUnknownType(t *testing.T) {
	type custom struct{ X int }
	kv := WithSpanAttributeKey("obj", custom{X: 1})
	if kv.Value.Type() != attribute.STRING {
		t.Errorf("expected string type for unknown value, got %v", kv.Value.Type())
	}
}

func TestWithSpanAttribute(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span")
	WithSpanAttribute(span, "custom.attr", "value")
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)
	if v, ok := attrs.Value("custom.attr"); !ok || v.AsString() != "value" {
		t.Errorf("expected custom.attr=value, got %v", v)
	}
}

func TestWithSpanAttributeNoopSpan(t *testing.T) {
	// A non-recording span (span from an unsampled context) should not panic.
	noopSpan := trace.SpanFromContext(context.Background())
	WithSpanAttribute(noopSpan, "custom.attr", "value")
}

func TestWithSpanAttributeKeyUnknownTypeValue(t *testing.T) {
	// Unknown types should be stringified via fmt.Sprintf, not return "value".
	type custom struct{ X int }
	kv := WithSpanAttributeKey("obj", custom{X: 1})
	if kv.Value.Type() != attribute.STRING {
		t.Errorf("expected string type for unknown value, got %v", kv.Value.Type())
	}
	if kv.Value.AsString() != "{1}" {
		t.Errorf("expected '{1}' for custom{X:1}, got %q", kv.Value.AsString())
	}
}

func TestStartSpanWithAttributes(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span",
		WithAttributes(attribute.String("custom.key", "custom.val")),
	)
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	attrs := attribute.NewSet(spans[0].Attributes...)
	if v, ok := attrs.Value("custom.key"); !ok || v.AsString() != "custom.val" {
		t.Errorf("expected custom.key=custom.val, got %v", v)
	}
}

func TestStartSpanDefaultSpanKind(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span")
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	if spans[0].SpanKind != trace.SpanKindInternal {
		t.Errorf("expected SpanKindInternal, got %v", spans[0].SpanKind)
	}
}

func TestStartSpanCustomSpanKind(t *testing.T) {
	rec, tp := newTestTracerProvider()
	defer tp.Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "test.tracer", "test.span",
		WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()
	span.End()

	spans := endedStubs(rec)
	if len(spans) != 1 {
		t.Fatalf("expected 1 exported span, got %d", len(spans))
	}
	if spans[0].SpanKind != trace.SpanKindClient {
		t.Errorf("expected SpanKindClient, got %v", spans[0].SpanKind)
	}
}

// TestActivityTransactionIDIsUUID ensures the transaction_id carried onto
// spans is a valid UUID (so it can be correlated across signals).
func TestActivityTransactionIDIsUUID(t *testing.T) {
	ctx := activity.NewContext("test")
	trxID, ok := activity.GetTransactionID(ctx)
	if !ok {
		t.Fatal("expected transaction ID to be set")
	}
	if err := uuid.Validate(trxID); err != nil {
		t.Fatalf("transaction ID %q is not a valid UUID: %v", trxID, err)
	}
}
