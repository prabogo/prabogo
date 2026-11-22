package masking

import (
	"encoding/json"
	"reflect"
	"strings"
)

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// DefaultSensitiveKeys is the canonical list of key patterns that are masked by
// default. Every log sink (stdout formatter and OTLP export) must mask the same
// set so a value can never leak through one sink while being hidden in another.
var DefaultSensitiveKeys = []string{
	"bearer_key",
	"bearerkey",
	"bearer",
	"password",
	"passwd",
	"secret",
	"token",
	"authorization",
	"api_key",
	"apikey",
	"access_token",
	"refresh_token",
	"credit_card",
	"card_number",
	"cvv",
}

var (
	// SensitiveKeys is kept for backwards compatibility; it mirrors
	// DefaultSensitiveKeys.
	SensitiveKeys = DefaultSensitiveKeys

	MaskValue = "***MASKED***"
)

func isSensitiveKey(key string) bool {
	for _, k := range SensitiveKeys {
		if strings.Contains(key, k) {
			return true
		}
	}
	return false
}

// IsSensitiveKey reports whether key matches any of the SensitiveKeys
// (case-insensitive). It is exported so callers can branch on sensitivity
// without comparing non-comparable values with !=.
func IsSensitiveKey(key string) bool {
	return isSensitiveKey(strings.ToLower(key))
}

func MaskValueIfSensitive(key string, value interface{}) interface{} {
	if isSensitiveKey(strings.ToLower(key)) {
		return MaskValue
	}
	return value
}

type maskRule func(key string) bool

func defaultRule(key string) bool {
	return isSensitiveKey(key)
}

func allowlistRule(allowlist []string) maskRule {
	return func(key string) bool {
		for _, allowed := range allowlist {
			if strings.EqualFold(key, strings.ToLower(allowed)) {
				return true
			}
		}
		return false
	}
}

// MaskData masks every sensitive key found anywhere in data using the default
// SensitiveKeys. It is the default entry point used by log sinks so masking
// stays consistent across stdout and OTLP.
func MaskData(data interface{}) interface{} {
	return maskData(data, defaultRule)
}

// MaskDataAllowlist masks only the keys contained in allowlist. Callers that
// need the default behaviour should use MaskData instead.
func MaskDataAllowlist(data interface{}, allowlist []string) interface{} {
	if len(allowlist) == 0 {
		return maskData(data, defaultRule)
	}
	return maskData(data, allowlistRule(allowlist))
}

func maskData(data interface{}, rule maskRule) interface{} {
	if data == nil {
		return data
	}

	value := reflect.ValueOf(data)
	switch value.Kind() {
	case reflect.Map:
		return maskMap(value, rule)
	case reflect.Slice, reflect.Array:
		return maskSlice(value, rule)
	case reflect.Struct:
		return maskStruct(value, rule)
	case reflect.Ptr:
		if value.IsNil() {
			return data
		}
		return maskData(value.Elem().Interface(), rule)
	default:
		return data
	}
}

func maskMap(value reflect.Value, rule maskRule) interface{} {
	result := make(map[string]interface{}, value.Len())
	for _, keyElem := range value.MapKeys() {
		key := keyElem.String()
		mapped := maskValueIfRule(key, value.MapIndex(keyElem).Interface(), rule)
		if s, ok := mapped.(string); ok && s == MaskValue {
			result[key] = MaskValue
			continue
		}
		result[key] = maskData(mapped, rule)
	}
	return result
}

func maskSlice(value reflect.Value, rule maskRule) interface{} {
	if value.Type().Elem().Kind() == reflect.Uint8 {
		// Byte slices/arrays (e.g. raw message payloads) are opaque; render them
		// as text instead of a list of numbers.
		return bytesToString(value)
	}
	result := make([]interface{}, value.Len())
	for i := 0; i < value.Len(); i++ {
		result[i] = maskData(value.Index(i).Interface(), rule)
	}
	return result
}

func bytesToString(value reflect.Value) string {
	if value.Kind() == reflect.Slice {
		return string(value.Bytes())
	}
	buf := make([]byte, value.Len())
	for i := 0; i < value.Len(); i++ {
		buf[i] = byte(value.Index(i).Uint())
	}
	return string(buf)
}

func maskStruct(value reflect.Value, rule maskRule) interface{} {
	// Types that define their own JSON representation (e.g. time.Time,
	// sql.NullString) keep it, otherwise they would be collapsed into an empty
	// object and lose their meaning in the logs.
	typeValue := value.Type()
	if typeValue.Implements(jsonMarshalerType) || reflect.PointerTo(typeValue).Implements(jsonMarshalerType) {
		return value.Interface()
	}

	result := make(map[string]interface{}, typeValue.NumField())
	for i := 0; i < typeValue.NumField(); i++ {
		field := typeValue.Field(i)
		if !value.Field(i).CanInterface() {
			continue
		}
		key := fieldKey(field)
		mapped := maskValueIfRule(key, value.Field(i).Interface(), rule)
		result[key] = maskData(mapped, rule)
	}
	return result
}

func maskValueIfRule(key string, value interface{}, rule maskRule) interface{} {
	if rule(strings.ToLower(key)) {
		return MaskValue
	}
	return value
}

func fieldKey(field reflect.StructField) string {
	if tag := field.Tag.Get("json"); tag != "" {
		if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
			return name
		}
	}
	if tag := field.Tag.Get("db"); tag != "" {
		if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
			return name
		}
	}
	return field.Name
}
