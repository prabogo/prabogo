package masking

import (
	"encoding/json"
	"testing"
	"time"
)

type sample struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	BearerKey string    `json:"bearer_key" db:"bearer_key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func TestMaskDataMasksNestedStructFields(t *testing.T) {
	input := []sample{{ID: 87, Name: "budiman_client", BearerKey: "secret", CreatedAt: time.Now()}}

	masked := MaskData(input)

	encoded, err := json.Marshal(masked)
	if err != nil {
		t.Fatalf("marshal masked data: %v", err)
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal masked data: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 element, got %d", len(decoded))
	}
	if decoded[0]["bearer_key"] != MaskValue {
		t.Errorf("bearer_key should be masked, got %#v", decoded[0]["bearer_key"])
	}
	if decoded[0]["name"] != "budiman_client" {
		t.Errorf("name should be preserved, got %#v", decoded[0]["name"])
	}
	if decoded[0]["id"] != float64(87) {
		t.Errorf("id should be preserved, got %#v", decoded[0]["id"])
	}
	if _, ok := decoded[0]["created_at"].(string); !ok {
		t.Errorf("time.Time should keep its RFC3339 form, got %#v", decoded[0]["created_at"])
	}
}

func TestMaskDataMasksMapValues(t *testing.T) {
	masked := MaskData(map[string]interface{}{
		"access_token": "abc",
		"nested":       map[string]interface{}{"authorization": "Bearer xyz"},
		"keep":         "value",
	})

	encoded, _ := json.Marshal(masked)
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["access_token"] != MaskValue {
		t.Errorf("access_token should be masked, got %#v", decoded["access_token"])
	}
	nested := decoded["nested"].(map[string]interface{})
	if nested["authorization"] != MaskValue {
		t.Errorf("nested authorization should be masked, got %#v", nested["authorization"])
	}
	if decoded["keep"] != "value" {
		t.Errorf("keep should be preserved, got %#v", decoded["keep"])
	}
}

func TestMaskDataConvertsByteSlices(t *testing.T) {
	if got := MaskData([]byte("hello")); got != "hello" {
		t.Errorf("byte slice should become a string, got %#v", got)
	}
	if got := MaskData([3]byte{'a', 'b', 'c'}); got != "abc" {
		t.Errorf("byte array should become a string, got %#v", got)
	}
}

func TestMaskDataEmptyAllowlistUsesDefaults(t *testing.T) {
	masked := MaskDataAllowlist(map[string]interface{}{"password": "hunter2"}, nil)
	encoded, _ := json.Marshal(masked)
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["password"] != MaskValue {
		t.Errorf("empty allowlist should fall back to defaults, got %#v", decoded["password"])
	}
}

func TestMaskDataAllowlistOnlyMasksListedKeys(t *testing.T) {
	masked := MaskDataAllowlist(map[string]interface{}{"password": "hunter2", "token": "t"}, []string{"token"})
	encoded, _ := json.Marshal(masked)
	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["token"] != MaskValue {
		t.Errorf("token should be masked, got %#v", decoded["token"])
	}
	if decoded["password"] != "hunter2" {
		t.Errorf("password is not in the allowlist and should be untouched, got %#v", decoded["password"])
	}
}

func TestMaskDataPassesThroughLeaves(t *testing.T) {
	if got := MaskData(nil); got != nil {
		t.Errorf("nil should stay nil, got %#v", got)
	}
	if got := MaskData("plain"); got != "plain" {
		t.Errorf("string should pass through, got %#v", got)
	}
	if got := MaskData(42); got != 42 {
		t.Errorf("int should pass through, got %#v", got)
	}
	var ptr *sample
	if got := MaskData(ptr); got != interface{}(ptr) {
		t.Errorf("nil pointer should be returned unchanged, got %#v", got)
	}
}
