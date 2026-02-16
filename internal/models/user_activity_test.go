package models

import (
	"encoding/json"
	"testing"
)

func TestUserActivityTableName(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("core_")
	if got := (UserActivity{}).TableName(); got != "core_user_activities" {
		t.Fatalf("expected table name core_user_activities, got %q", got)
	}
}

func TestUintSliceScanValue(t *testing.T) {
	var s UintSlice
	if err := s.Scan(nil); err != nil {
		t.Fatalf("unexpected error on Scan(nil): %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil slice after Scan(nil), got %v", s)
	}

	raw := []uint{1, 2, 3}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}

	if err := s.Scan(encoded); err != nil {
		t.Fatalf("unexpected error on Scan: %v", err)
	}
	if len(s) != len(raw) || s[0] != 1 || s[2] != 3 {
		t.Fatalf("unexpected scan result: %v", s)
	}

	if _, err := (UintSlice(nil)).Value(); err != nil {
		t.Fatalf("unexpected error on Value(nil): %v", err)
	}

	val, err := s.Value()
	if err != nil {
		t.Fatalf("unexpected error on Value: %v", err)
	}
	if string(val.([]byte)) == "" {
		t.Fatalf("expected non-empty JSON value")
	}

	if err := s.Scan("invalid"); err == nil {
		t.Fatalf("expected error on Scan with invalid type")
	}
}
