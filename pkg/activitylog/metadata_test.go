package activitylog

import (
	"encoding/json"
	"testing"
)

func TestMergeMetadataNilPatch(t *testing.T) {
	existing := `{"a":1}`
	merged := MergeMetadata(&existing, nil)
	if merged == nil || *merged != existing {
		t.Fatalf("expected existing metadata to be returned unchanged")
	}
}

func TestMergeMetadataNoExisting(t *testing.T) {
	patch := map[string]interface{}{
		"serviceName": "PaymentService",
		"status":      "SUCCESS",
	}
	merged := MergeMetadata(nil, patch)
	if merged == nil {
		t.Fatalf("expected merged metadata")
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(*merged), &payload); err != nil {
		t.Fatalf("expected valid metadata json: %v", err)
	}
	if payload["serviceName"] != "PaymentService" {
		t.Fatalf("expected serviceName to be set")
	}
}

func TestMergeMetadataWithExistingJSON(t *testing.T) {
	existing := `{"source":"SERVICE","operationTrail":[{"name":"A"}]}`
	patch := map[string]interface{}{
		"serviceStatus": "SUCCESS",
	}
	merged := MergeMetadata(&existing, patch)
	if merged == nil {
		t.Fatalf("expected merged metadata")
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(*merged), &payload); err != nil {
		t.Fatalf("expected valid metadata json: %v", err)
	}
	if payload["source"] != "SERVICE" {
		t.Fatalf("expected existing source to be preserved")
	}
	if payload["serviceStatus"] != "SUCCESS" {
		t.Fatalf("expected patch to be applied")
	}
}

func TestMergeMetadataWithExistingRawString(t *testing.T) {
	existing := "plain-text"
	patch := map[string]interface{}{
		"serviceStatus": "ERROR",
	}
	merged := MergeMetadata(&existing, patch)
	if merged == nil {
		t.Fatalf("expected merged metadata")
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(*merged), &payload); err != nil {
		t.Fatalf("expected valid metadata json: %v", err)
	}
	if payload["rawMetadata"] != "plain-text" {
		t.Fatalf("expected raw metadata to be preserved")
	}
	if payload["serviceStatus"] != "ERROR" {
		t.Fatalf("expected patch to be applied")
	}
}
