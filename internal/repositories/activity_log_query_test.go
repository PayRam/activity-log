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
	var activities []models.ActivityLog
	stmt := query.Find(&activities).Statement
	return stmt.SQL.String()
}

func TestApplyActivityLogGetFilters_Basic(t *testing.T) {
	db := newDryRunDB(t)
	session := "sess"
	search := "term"
	filter := ActivityLogFilters{
		IDS:             []uint{1, 2},
		Methods:         []string{"GET"},
		MemberIDs:       []uint{10},
		APIStatuses:     []string{"SUCCESS"},
		EventNames:      []string{"LOGIN"},
		EventCategories: []string{"AUTH"},
		StatusCodes:     []int{200, 201},
		SessionIDs:      []string{session},
		Search:          &search,
		IPAddresses:     []string{"1.1.1.1"},
		Countries:       []string{"US"},
		Roles:           []string{"admin"},
		ProjectIDs:      []uint{7},
	}

	query := ApplyActivityLogGetFilters(db.Model(&models.ActivityLog{}), filter)
	sql := buildSQL(t, query)

	expected := []string{
		"IN", "method", "member_id", "api_status", "event_name", "event_category",
		"status_code", "session_id", "ip_address", "country", "role",
		"SELECT id FROM members",
		"project_ids::jsonb @>",
	}
	for _, part := range expected {
		if !strings.Contains(sql, part) {
			t.Fatalf("expected SQL to contain %q, got: %s", part, sql)
		}
	}
}

func TestApplyActivityLogGetFilters_ExcludeMethodsAndAPIStatuses(t *testing.T) {
	db := newDryRunDB(t)
	filter := ActivityLogFilters{
		ExcludeMethods:     []string{"READ"},
		ExcludeAPIStatuses: []string{"SUCCESS"},
	}

	query := ApplyActivityLogGetFilters(db.Model(&models.ActivityLog{}), filter)
	sql := buildSQL(t, query)

	if !strings.Contains(sql, "method NOT IN") {
		t.Fatalf("expected SQL to contain method NOT IN, got: %s", sql)
	}
	if !strings.Contains(sql, "api_status NOT IN") {
		t.Fatalf("expected SQL to contain api_status NOT IN, got: %s", sql)
	}
}

func TestApplyActivityLogGetFilters_ExcludeActionStatusPairs(t *testing.T) {
	db := newDryRunDB(t)
	filter := ActivityLogFilters{
		ExcludeActionStatusPairs: []ActionStatusPairFilter{{APIAction: "READ", APIStatus: "SUCCESS"}},
	}

	query := ApplyActivityLogGetFilters(db.Model(&models.ActivityLog{}), filter)
	sql := buildSQL(t, query)

	if !strings.Contains(sql, "NOT (") || !strings.Contains(sql, "api_action") || !strings.Contains(sql, "api_status") {
		t.Fatalf("expected SQL to contain grouped api_action/api_status exclusion, got: %s", sql)
	}
}

func TestApplyActivityLogGetFilters_ProjectIDsModes(t *testing.T) {
	db := newDryRunDB(t)
	filter := ActivityLogFilters{ProjectIDs: []uint{}}

	query := ApplyActivityLogGetFilters(db.Model(&models.ActivityLog{}), filter)
	sql := buildSQL(t, query)
	if !strings.Contains(sql, "project_ids IS NULL") {
		t.Fatalf("expected SQL to contain NO_IDS condition, got: %s", sql)
	}

	filter = ActivityLogFilters{ProjectIDs: []uint{7}}
	query = ApplyActivityLogGetFilters(db.Model(&models.ActivityLog{}), filter)
	sql = buildSQL(t, query)
	if !strings.Contains(sql, "project_ids::jsonb @>") {
		t.Fatalf("expected SQL to contain project_ids containment condition, got: %s", sql)
	}
	if !strings.Contains(sql, "project_ids IS NULL") {
		t.Fatalf("expected SQL to include no-project clause, got: %s", sql)
	}
}

func TestApplyActivityLogsPaginationConditions(t *testing.T) {
	db := newDryRunDB(t)
	limit := 10
	offset := 5
	sortBy := "created_at"
	order := "ASC"
	gt := uint(5)
	lt := uint(20)
	createdAfter := time.Now().Add(-time.Hour)

	filter := ActivityLogFilters{
		Limit:         &limit,
		Offset:        &offset,
		SortBy:        &sortBy,
		Order:         &order,
		GreaterThanID: &gt,
		LessThanID:    &lt,
		CreatedAfter:  &createdAfter,
	}

	query := ApplyActivityLogsPaginationConditions(db.Model(&models.ActivityLog{}), filter)
	sql := buildSQL(t, query)
	if !strings.Contains(sql, "ORDER BY created_at ASC") {
		t.Fatalf("expected ORDER BY created_at ASC, got: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT") || !strings.Contains(sql, "OFFSET") {
		t.Fatalf("expected LIMIT/OFFSET in SQL, got: %s", sql)
	}

	invalidSort := "bad_column"
	filter.SortBy = &invalidSort
	query = ApplyActivityLogsPaginationConditions(db.Model(&models.ActivityLog{}), filter)
	sql = buildSQL(t, query)
	if !strings.Contains(sql, "ORDER BY id ASC") {
		t.Fatalf("expected ORDER BY id ASC for invalid column, got: %s", sql)
	}
}
