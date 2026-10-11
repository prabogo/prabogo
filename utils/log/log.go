package log

import (
	"context"

	"github.com/sirupsen/logrus"

	"prabogo/utils/activity"
)

const (
	MAX_LOG_ENTRY_SIZE = 8 * 1024
)

type Trail struct {
	Label   string
	Payload interface{}
}

// WithContext returns a logrus entry carrying the fields from ctx (activity
// context). Trace and span IDs are NOT added here — the OTel Logs SDK
// (OTelLogrusHook in otel_hook.go) populates the standard OTLP
// LogRecord.trace_id / LogRecord.span_id fields automatically from the span
// context in entry.Context. Adding them again as logrus fields would duplicate
// them in the log body (attributes.trace_id / attributes.span_id), which is
// redundant and non-standard.
//
// Grafana correlation relies on the SDK's top-level fields:
//   - Tempo tracesToLogs (filterByTraceID) → Loki | json | traceid="..."
//   - Loki derivedFields → regex on the top-level "traceid" field in the body
func WithContext(ctx context.Context) *logrus.Entry {
	fields := activity.GetFields(ctx)

	return logrus.WithContext(ctx).WithFields(fields)
}

func LogOrmer(obj interface{}, prefix string) {
	logrus.Debug(prefix, obj)
}

func LogTrail(obj interface{}, prefix string) {
	logrus.WithFields(logrus.Fields{
		"data": obj,
	}).Info(prefix)
}

func LogTrails(trails []Trail) {
	for _, trail := range trails {
		go LogTrail(trail.Payload, trail.Label)
	}
}
