package log

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"

	"prabogo/utils/masking"
)

const (
	LoggerName = "prabogo"
)

// OTelLogrusHook forwards logrus entries to the OpenTelemetry Logs SDK.
//
// Field masking is performed by MaskingHook, which runs before this hook and
// rewrites entry.Data in place. This hook still guards sensitive keys itself so
// the OTLP sink stays safe even if it is registered without MaskingHook.
type OTelLogrusHook struct {
	logger log.Logger
}

func NewOTelLogrusHook() *OTelLogrusHook {
	return &OTelLogrusHook{
		logger: global.GetLoggerProvider().Logger(LoggerName),
	}
}

func (h *OTelLogrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *OTelLogrusHook) Fire(entry *logrus.Entry) error {
	record := log.Record{}

	record.SetTimestamp(entry.Time)
	record.SetBody(attribute.StringValue(entry.Message))
	record.SetSeverity(severityFromLogrus(entry.Level))
	record.SetSeverityText(entry.Level.String())

	// Always mask before export so the OTLP record can never carry a value the
	// stdout sink redacts.
	for k, v := range entry.Data {
		if masking.IsSensitiveKey(k) {
			record.AddAttributes(attribute.String(k, masking.MaskValue))
			continue
		}
		record.AddAttributes(attribute.String(k, maskFieldValue(v)))
	}

	emitCtx := entry.Context
	if emitCtx == nil {
		emitCtx = context.Background()
	}

	h.logger.Emit(emitCtx, record)
	return nil
}

func maskFieldValue(value interface{}) string {
	// Fast path for string values — avoid reflection-based masking.
	if s, ok := value.(string); ok {
		return s
	}
	// For non-scalar values (maps, slices, structs), apply masking recursively
	// to redact nested sensitive fields.
	return stringifyValue(masking.MaskData(value))
}

func stringifyValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return masking.MaskValue
	}
	return string(b)
}

func severityFromLogrus(level logrus.Level) log.Severity {
	switch level {
	case logrus.PanicLevel, logrus.FatalLevel:
		return log.SeverityFatal
	case logrus.ErrorLevel:
		return log.SeverityError
	case logrus.WarnLevel:
		return log.SeverityWarn
	case logrus.InfoLevel:
		return log.SeverityInfo
	case logrus.DebugLevel:
		return log.SeverityDebug
	default:
		return log.SeverityTrace
	}
}
