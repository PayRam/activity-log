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

	if got := GetTableName(DefaultUserActivityTableName); got != "ua_user_activities" {
		t.Fatalf("expected table name ua_user_activities, got %q", got)
	}
}

func TestCustomUserActivityTableName(t *testing.T) {
	ResetTablePrefix()
	t.Cleanup(ResetTablePrefix)

	SetTablePrefix("core_")
	SetUserActivityTableName("activity_logs")

	if got := GetUserActivityTableName(); got != "activity_logs" {
		t.Fatalf("expected user activity table name activity_logs, got %q", got)
	}

	if got := GetTableName(DefaultUserActivityTableName); got != "core_activity_logs" {
		t.Fatalf("expected full table name core_activity_logs, got %q", got)
	}
}
