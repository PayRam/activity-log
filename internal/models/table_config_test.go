package models

import "testing"

func TestTablePrefix(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("ua_")
	if got := GetTablePrefix(); got != "ua_" {
		t.Fatalf("expected table prefix ua_, got %q", got)
	}

	// SetTablePrefix should only apply once.
	SetTablePrefix("other_")
	if got := GetTablePrefix(); got != "ua_" {
		t.Fatalf("expected table prefix to remain ua_, got %q", got)
	}

	if got := GetTableName(DefaultActivityLogTableName); got != "ua_activity_logs" {
		t.Fatalf("expected table name ua_activity_logs, got %q", got)
	}
}

func TestCustomActivityLogTableName(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("core_")
	SetActivityLogTableName("activity_logs")

	if got := GetActivityLogTableName(); got != "activity_logs" {
		t.Fatalf("expected activity log table name activity_logs, got %q", got)
	}

	if got := GetTableName(DefaultActivityLogTableName); got != "core_activity_logs" {
		t.Fatalf("expected full table name core_activity_logs, got %q", got)
	}
}
