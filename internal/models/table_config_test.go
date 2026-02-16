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

	if got := GetTableName("user_activities"); got != "ua_user_activities" {
		t.Fatalf("expected table name ua_user_activities, got %q", got)
	}
}
