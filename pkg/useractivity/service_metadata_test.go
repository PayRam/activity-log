package useractivity

import (
	"encoding/json"
	"testing"
)

func TestMarshalServiceMetadataSourceOnly(t *testing.T) {
	metadata := marshalServiceMetadata(ServiceMetadata{Source: "SERVICE"})
	if metadata == nil {
		t.Fatalf("expected source-only metadata to be preserved")
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(*metadata), &payload); err != nil {
		t.Fatalf("expected valid metadata json: %v", err)
	}
	if payload["source"] != "SERVICE" {
		t.Fatalf("expected source field to be present, got %v", payload["source"])
	}
}

func TestMergeServiceMetadataSourceOnly(t *testing.T) {
	base := `{"workflow":"payment"}`
	metadata := mergeServiceMetadata(&base, ServiceMetadata{Source: "SERVICE"})
	if metadata == nil {
		t.Fatalf("expected merged metadata")
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(*metadata), &payload); err != nil {
		t.Fatalf("expected valid metadata json: %v", err)
	}
	if payload["source"] != "SERVICE" {
		t.Fatalf("expected source field to be present after merge")
	}
	if payload["workflow"] != "payment" {
		t.Fatalf("expected existing metadata to be preserved")
	}
}
