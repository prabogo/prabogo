package log

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// withTraceParent returns a context carrying a span so we can verify that
// WithContext does NOT duplicate trace/span IDs as logrus fields. The OTel
// Logs SDK (OTelLogrusHook) populates the standard OTLP LogRecord.trace_id /
// LogRecord.span_id fields automatically from the span context; adding them
// again here would be redundant.
func withTraceParent(t *testing.T) context.Context {
	t.Helper()
	tid, err := trace.TraceIDFromHex("0abc0123456789abcdef0123456789a1")
	if err != nil {
		t.Fatalf("TraceIDFromHex: %v", err)
	}
	sid, err := trace.SpanIDFromHex("0def0123456789a1")
	if err != nil {
		t.Fatalf("SpanIDFromHex: %v", err)
	}
	sctx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sctx)
}

func TestWithContextDoesNotDuplicateTraceIDs(t *testing.T) {
	entry := WithContext(withTraceParent(t))

	// WithContext must NOT add trace_id/span_id as logrus fields — the OTel
	// Logs SDK populates the standard OTLP fields automatically.
	if _, ok := entry.Data["trace_id"]; ok {
		t.Errorf("trace_id should not be set by WithContext (SDK handles it), got %v", entry.Data["trace_id"])
	}
	if _, ok := entry.Data["span_id"]; ok {
		t.Errorf("span_id should not be set by WithContext (SDK handles it), got %v", entry.Data["span_id"])
	}
}

func TestWithContextNoTraceLeavesNoIDs(t *testing.T) {
	entry := WithContext(context.Background())
	if _, ok := entry.Data["trace_id"]; ok {
		t.Errorf("expected no trace_id when ctx has no span, got %v", entry.Data["trace_id"])
	}
	if _, ok := entry.Data["span_id"]; ok {
		t.Errorf("expected no span_id when ctx has no span, got %v", entry.Data["span_id"])
	}
}
