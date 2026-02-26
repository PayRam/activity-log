package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestActivityLogTableName(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("core_")
	if got := (ActivityLog{}).TableName(); got != "core_activity_logs" {
		t.Fatalf("expected table name core_activity_logs, got %q", got)
	}
}

func TestActivityLogCustomTableName(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("core_")
	SetActivityLogTableName("activity_logs")
	if got := (ActivityLog{}).TableName(); got != "core_activity_logs" {
		t.Fatalf("expected table name core_activity_logs, got %q", got)
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

	if err := s.Scan(string(encoded)); err != nil {
		t.Fatalf("unexpected error on Scan(string): %v", err)
	}
	if len(s) != len(raw) || s[1] != 2 {
		t.Fatalf("unexpected scan result from string input: %v", s)
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
		t.Fatalf("expected error on Scan with invalid JSON string")
	}

	if err := s.Scan(123); err == nil {
		t.Fatalf("expected error on Scan with unsupported type")
	} else if !strings.Contains(err.Error(), "UintSlice.Scan") {
		t.Fatalf("expected error to mention UintSlice.Scan, got %v", err)
	}
}
