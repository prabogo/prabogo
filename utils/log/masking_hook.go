package log

import (
	"github.com/sirupsen/logrus"

	"prabogo/utils/masking"
)

// MaskingHook masks sensitive fields in-place on every log entry before it is
// formatted or exported.
//
// logrus fires hooks on the entry returned by Entry.Dup() and then serializes
// that same entry with the configured formatter (see logrus.entry.log). A hook
// that only masks the copy it receives without writing the result back would
// therefore leave the original values in the stdout output. This hook assigns
// the masked values back to entry.Data so that both the stdout formatter and
// any hook registered after it (e.g. the OTLP hook) see masked data.
//
// It must be registered before other log-emitting hooks so that no sink can
// observe the unmasked values.
type MaskingHook struct {
	// Keys overrides the masked key patterns. When empty,
	// masking.DefaultSensitiveKeys is used.
	Keys []string
}

func NewMaskingHook() *MaskingHook {
	return &MaskingHook{}
}

func (h *MaskingHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *MaskingHook) Fire(entry *logrus.Entry) error {
	for k, v := range entry.Data {
		if masking.IsSensitiveKey(k) {
			entry.Data[k] = masking.MaskValue
			continue
		}
		// Store the structured, masked value so json-encoding formatters keep
		// the original shape while sensitive values stay redacted.
		entry.Data[k] = h.maskValue(v)
	}
	return nil
}

func (h *MaskingHook) maskValue(value interface{}) interface{} {
	switch v := value.(type) {
	case nil:
		return nil
	case string, bool, float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return value
	case error:
		return v.Error()
	}
	if len(h.Keys) > 0 {
		return masking.MaskDataAllowlist(value, h.Keys)
	}
	return masking.MaskData(value)
}
