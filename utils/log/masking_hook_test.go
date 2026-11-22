package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"prabogo/utils/masking"
)

var errFailed = errors.New("boom")

type maskFixture struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	BearerKey string    `json:"bearer_key" db:"bearer_key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

const secret = "acb2ba897d8febd52783c72b9573cf60bf2cfb211004f5bf65"

// newTestLogger returns a logrus logger that mimics configureLogging(): the
// source-context hook is replaced by the masking hook (registered first) so the
// hook ordering under test matches production.
func newTestLogger(out *bytes.Buffer) *logrus.Logger {
	logger := logrus.New()
	logger.Out = out
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.AddHook(&MaskingHook{})
	return logger
}

// TestMaskingHookMasksStdoutPayload is the regression test for the original
// bug: the OTLP hook masked its copy of the entry, but never wrote the masked
// values back, so the stdout formatter still serialized the raw secret.
func TestMaskingHookMasksStdoutPayload(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	logger.WithFields(logrus.Fields{
		"action": "message_client_upsert",
		"payload": []maskFixture{
			{ID: 87, Name: "budiman_client", BearerKey: secret, CreatedAt: time.Now()},
		},
	}).Info("message client upsert success")

	line := out.String()
	if strings.Contains(line, secret) {
		t.Fatalf("stdout leaked the bearer key: %s", line)
	}
	if !strings.Contains(line, masking.MaskValue) {
		t.Fatalf("stdout did not contain the masked placeholder: %s", line)
	}

	// The remaining fields must survive intact so the log stays useful.
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &decoded); err != nil {
		t.Fatalf("unmarshal stdout line: %v", err)
	}
	payload, ok := decoded["payload"].([]interface{})
	if !ok || len(payload) != 1 {
		t.Fatalf("expected payload array with one entry, got %#v", decoded["payload"])
	}
	first := payload[0].(map[string]interface{})
	if first["name"] != "budiman_client" {
		t.Errorf("name should be preserved, got %#v", first["name"])
	}
	if first["bearer_key"] != masking.MaskValue {
		t.Errorf("bearer_key should be masked, got %#v", first["bearer_key"])
	}
}

// TestMaskingHookMasksBothSinks ensures the same entry is masked in the stdout
// output and in the OTLP record, which is the contract the fix must uphold.
func TestMaskingHookMasksBothSinks(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	otelHook, cl := newCapturingHook()
	logger.AddHook(otelHook)

	logger.WithFields(logrus.Fields{
		"payload": []maskFixture{{ID: 87, Name: "budiman_client", BearerKey: secret}},
		"result":  []maskFixture{{ID: 87, Name: "budiman_client", BearerKey: secret}},
	}).Info("message client upsert success")

	if line := out.String(); strings.Contains(line, secret) {
		t.Fatalf("stdout leaked the bearer key: %s", line)
	}
	if len(cl.records) != 1 {
		t.Fatalf("expected 1 OTLP record, got %d", len(cl.records))
	}
	cl.records[0].WalkAttributes(func(kv attribute.KeyValue) bool {
		if strings.Contains(kv.Value.AsString(), secret) {
			t.Errorf("OTLP attribute %q leaked the bearer key", kv.Key)
		}
		return true
	})
}

func TestMaskingHookMasksTopLevelSensitiveKey(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	logger.WithField("password", "hunter2").Info("login")

	if strings.Contains(out.String(), "hunter2") {
		t.Fatalf("stdout leaked the password: %s", out.String())
	}
}

func TestMaskingHookPreservesScalarTypes(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	logger.WithFields(logrus.Fields{
		"count": 42,
		"ok":    true,
		"ratio": 0.5,
	}).Info("scalars")

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &decoded); err != nil {
		t.Fatalf("unmarshal stdout line: %v", err)
	}
	if decoded["count"] != float64(42) {
		t.Errorf("count should stay numeric, got %#v", decoded["count"])
	}
	if decoded["ok"] != true {
		t.Errorf("ok should stay boolean, got %#v", decoded["ok"])
	}
	if decoded["ratio"] != 0.5 {
		t.Errorf("ratio should stay numeric, got %#v", decoded["ratio"])
	}
}

// TestMaskingHookKeepsLogrusErrorField guards against breaking logrus' built-in
// error handling: WithError stores an error value that json-encoding cannot
// marshal, so the hook must convert it to its message.
func TestMaskingHookKeepsLogrusErrorField(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	logger.WithError(errFailed).Info("with error")

	if strings.Contains(out.String(), "unsupported type") {
		t.Fatalf("error field was not serialized: %s", out.String())
	}
	if !strings.Contains(out.String(), "boom") {
		t.Fatalf("error message missing from output: %s", out.String())
	}
}

// TestMaskingHookOutputFormatMatchesFormatter verifies the hook does not change
// how a non-masked record is rendered.
func TestMaskingHookDoesNotAlterUnrelatedFields(t *testing.T) {
	var out bytes.Buffer
	logger := newTestLogger(&out)

	logger.WithField("action", "http_client_upsert").Info("plain")

	if !strings.Contains(out.String(), `"action":"http_client_upsert"`) {
		t.Fatalf("unrelated field was altered: %s", out.String())
	}
	if !strings.Contains(out.String(), `"msg":"plain"`) {
		t.Fatalf("message was altered: %s", out.String())
	}
}
