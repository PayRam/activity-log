package useractivity

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactJSONKeys(t *testing.T) {
	payload := []byte(`{"password":"secret","authorization":"Bearer abc.def.ghi","profile":{"token":"abc","name":"john"},"items":[{"apiKey":"k1"},{"value":1}]}`)
	redacted := RedactJSONKeys(payload, "password", "token", "apiKey", "authorization")

	if string(redacted) == string(payload) {
		t.Fatalf("expected redacted payload to differ")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(redacted, &decoded); err != nil {
		t.Fatalf("failed to unmarshal redacted json: %v", err)
	}

	if decoded["password"] != "***REDACTED***" {
		t.Fatalf("expected password to be redacted")
	}
	if decoded["authorization"] != "***REDACTED***" {
		t.Fatalf("expected authorization to be redacted")
	}
	profile := decoded["profile"].(map[string]interface{})
	if profile["token"] != "***REDACTED***" {
		t.Fatalf("expected token to be redacted")
	}
	items := decoded["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if first["apiKey"] != "***REDACTED***" {
		t.Fatalf("expected apiKey to be redacted")
	}
}

func TestRedactJSONKeysInvalidPayload(t *testing.T) {
	payload := []byte("not-json")
	redacted := RedactJSONKeys(payload, "password")
	if string(redacted) != string(payload) {
		t.Fatalf("expected invalid json to be returned unchanged")
	}
}

func TestRedactDefaultJSONKeys(t *testing.T) {
	payload := []byte(`{"password":"secret","access_token":"t","authorization":"Bearer abc","data":"ok"}`)
	redacted := RedactDefaultJSONKeys(payload)
	if !strings.Contains(string(redacted), "***REDACTED***") {
		t.Fatalf("expected default redaction to apply")
	}
}

func TestRedactDefaultJSONKeysCaseInsensitiveVariants(t *testing.T) {
	payload := []byte(`{
		"accessToken":"a",
		"refreshToken":"b",
		"Authorization":"Bearer x.y.z",
		"clientSecret":"c",
		"nested":{"Private-Key":"k","ID_TOKEN":"id"},
		"ok":"v"
	}`)

	redacted := RedactDefaultJSONKeys(payload)

	var decoded map[string]interface{}
	if err := json.Unmarshal(redacted, &decoded); err != nil {
		t.Fatalf("failed to unmarshal redacted json: %v", err)
	}

	if decoded["accessToken"] != "***REDACTED***" {
		t.Fatalf("expected accessToken to be redacted")
	}
	if decoded["refreshToken"] != "***REDACTED***" {
		t.Fatalf("expected refreshToken to be redacted")
	}
	if decoded["Authorization"] != "***REDACTED***" {
		t.Fatalf("expected Authorization to be redacted")
	}
	if decoded["clientSecret"] != "***REDACTED***" {
		t.Fatalf("expected clientSecret to be redacted")
	}

	nested := decoded["nested"].(map[string]interface{})
	if nested["Private-Key"] != "***REDACTED***" {
		t.Fatalf("expected Private-Key to be redacted")
	}
	if nested["ID_TOKEN"] != "***REDACTED***" {
		t.Fatalf("expected ID_TOKEN to be redacted")
	}
	if decoded["ok"] != "v" {
		t.Fatalf("expected non-sensitive field to remain unchanged")
	}
}

func TestRedactJSONKeysCaseInsensitive(t *testing.T) {
	payload := []byte(`{"secretKey":"v1","nested":{"SECRET_KEY":"v2"},"ok":"x"}`)
	redacted := RedactJSONKeys(payload, "SeCrEt_Key")

	var decoded map[string]interface{}
	if err := json.Unmarshal(redacted, &decoded); err != nil {
		t.Fatalf("failed to unmarshal redacted json: %v", err)
	}
	if decoded["secretKey"] != "***REDACTED***" {
		t.Fatalf("expected secretKey to be redacted")
	}
	nested := decoded["nested"].(map[string]interface{})
	if nested["SECRET_KEY"] != "***REDACTED***" {
		t.Fatalf("expected nested SECRET_KEY to be redacted")
	}
	if decoded["ok"] != "x" {
		t.Fatalf("expected non-sensitive field to remain unchanged")
	}
}
