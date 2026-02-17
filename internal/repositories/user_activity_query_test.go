package repositories

import (
	"strings"
	"testing"
	"time"

	"github.com/PayRam/activity-log/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		postgres.Open("host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"),
		&gorm.Config{
			DryRun:               true,
			DisableAutomaticPing: true,
		},
	)
	if err != nil {
		t.Fatalf("failed to open postgres in dry-run mode: %v", err)
	}
	return db
}

func buildSQL(t *testing.T, query *gorm.DB) string {
	t.Helper()
	var activities []models.UserActivity
	stmt := query.Find(&activities).Statement
	return stmt.SQL.String()
}

func TestApplyUserActivityGetFilters_Basic(t *testing.T) {
	db := newDryRunDB(t)
	session := "sess"
	search := "term"
	filter := UserActivityFilters{
		IDS:                 []uint{1, 2},
		Methods:             []string{"GET"},
		MemberIDs:           []uint{10},
		APIStatuses:         []string{"SUCCESS"},
		EventNames:          []string{"LOGIN"},
		EventCategories:     []string{"AUTH"},
		StatusCodes:         []int{200, 201},
		SessionID:           &session,
		Search:              &search,
		IPAddresses:         []string{"1.1.1.1"},
		Countries:           []string{"US"},
		Roles:               []string{"admin"},
		ExternalPlatformIDs: []uint{7},
	}

	query := ApplyUserActivityGetFilters(db.Model(&models.UserActivity{}), filter)
	sql := buildSQL(t, query)

	expected := []string{
		"IN", "method", "member_id", "api_status", "event_name", "event_category",
		"status_code", "session_id", "ip_address", "country", "role",
		"SELECT id FROM members",
		"external_platform_ids::jsonb @>",
	}
	for _, part := range expected {
		if !strings.Contains(sql, part) {
			t.Fatalf("expected SQL to contain %q, got: %s", part, sql)
		}
	}
}

func TestApplyUserActivityGetFilters_ProjectFilter(t *testing.T) {
	db := newDryRunDB(t)
	projectFilter := "NO_IDS"
	filter := UserActivityFilters{ProjectFilter: &projectFilter}

	query := ApplyUserActivityGetFilters(db.Model(&models.UserActivity{}), filter)
	sql := buildSQL(t, query)
	if !strings.Contains(sql, "external_platform_ids IS NULL") {
		t.Fatalf("expected SQL to contain NO_IDS condition, got: %s", sql)
	}

	projectFilter = "ALL"
	filter = UserActivityFilters{ProjectFilter: &projectFilter}
	query = ApplyUserActivityGetFilters(db.Model(&models.UserActivity{}), filter)
	sql = buildSQL(t, query)
	if !strings.Contains(sql, "external_platform_ids IS NOT NULL") {
		t.Fatalf("expected SQL to contain ALL condition, got: %s", sql)
	}
}

func TestApplyUserActivitiesPaginationConditions(t *testing.T) {
	db := newDryRunDB(t)
	limit := 10
	offset := 5
	sortBy := "created_at"
	order := "ASC"
	gt := uint(5)
	lt := uint(20)
	createdAfter := time.Now().Add(-time.Hour)

	filter := UserActivityFilters{
		Limit:         &limit,
		Offset:        &offset,
		SortBy:        &sortBy,
		Order:         &order,
		GreaterThanID: &gt,
		LessThanID:    &lt,
		CreatedAfter:  &createdAfter,
	}

	query := ApplyUserActivitiesPaginationConditions(db.Model(&models.UserActivity{}), filter)
	sql := buildSQL(t, query)
	if !strings.Contains(sql, "ORDER BY created_at ASC") {
		t.Fatalf("expected ORDER BY created_at ASC, got: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT") || !strings.Contains(sql, "OFFSET") {
		t.Fatalf("expected LIMIT/OFFSET in SQL, got: %s", sql)
	}

	invalidSort := "bad_column"
	filter.SortBy = &invalidSort
	query = ApplyUserActivitiesPaginationConditions(db.Model(&models.UserActivity{}), filter)
	sql = buildSQL(t, query)
	if !strings.Contains(sql, "ORDER BY id ASC") {
		t.Fatalf("expected ORDER BY id ASC for invalid column, got: %s", sql)
	}
}
