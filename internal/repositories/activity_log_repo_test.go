package repositories

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/PayRam/activity-log/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ACTIVITY_LOG_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set ACTIVITY_LOG_TEST_POSTGRES_DSN to run postgres repository tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open postgres: %v", err)
	}
	if err := db.AutoMigrate(&models.ActivityLog{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	tableName := models.GetTableName(models.DefaultActivityLogTableName)
	if err := db.Exec("TRUNCATE TABLE " + tableName + " RESTART IDENTITY").Error; err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}
	return db
}

func TestActivityLogRepositoryCreate(t *testing.T) {
	db := newPostgresDB(t)
	repo := NewActivityLogRepository(db, zap.NewNop())

	params := CreateActivityLogParams{
		SessionID: "sess-1",
		Method:    "GET",
		APIPart:   "/ping",
		APIStatus: "SUCCESS",
		APIAction: "READ",
	}
	created, err := repo.CreateActivityLogs(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created ID to be set")
	}
}

func TestActivityLogRepositoryListAndCategories(t *testing.T) {
	db := newPostgresDB(t)
	repo := NewActivityLogRepository(db, zap.NewNop())

	records := []CreateActivityLogParams{
		{SessionID: "s1", Method: "GET", APIPart: "/a", APIStatus: "SUCCESS", APIAction: "READ", EventCategory: strPtr("AUTH")},
		{SessionID: "s2", Method: "POST", APIPart: "/b", APIStatus: "SUCCESS", APIAction: "WRITE", EventCategory: strPtr("PAYMENT")},
		{SessionID: "s3", Method: "GET", APIPart: "/c", APIStatus: "SUCCESS", APIAction: "READ", EventCategory: strPtr("AUTH")},
	}
	for i := range records {
		if _, err := repo.CreateActivityLogs(context.Background(), records[i]); err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	filter := ActivityLogFilters{Methods: []string{"GET"}}
	list, total, err := repo.GetActivityLogs(context.Background(), filter)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected list length 2, got %d", len(list))
	}

	categories, err := repo.GetEventCategories(context.Background())
	if err != nil {
		t.Fatalf("categories error: %v", err)
	}
	if len(categories) != 2 || categories[0] != "AUTH" || categories[1] != "PAYMENT" {
		t.Fatalf("unexpected categories: %v", categories)
	}
}

func TestActivityLogRepositoryUpdateMissingRecord(t *testing.T) {
	db := newPostgresDB(t)
	repo := NewActivityLogRepository(db, zap.NewNop())

	params := UpdateActivityLogSessionParams{SessionID: "missing-session"}
	if _, err := repo.UpdateActivityLogSessionID(context.Background(), params); err == nil {
		t.Fatalf("expected error for missing session_id record")
	}
}

func TestActivityLogRepositoryUpdateDryRun(t *testing.T) {
	db := newPostgresDB(t).Session(&gorm.Session{DryRun: true})
	repo := NewActivityLogRepository(db, zap.NewNop())

	status := "SUCCESS"
	action := "READ"
	method := "GET"
	apiPart := "/ping"
	params := UpdateActivityLogSessionParams{
		SessionID: "sess-1",
		APIStatus: &status,
		APIAction: &action,
		Method:    &method,
		APIPart:   &apiPart,
	}
	if _, err := repo.UpdateActivityLogSessionID(context.Background(), params); err != nil {
		t.Fatalf("expected no error in dry run, got %v", err)
	}
}

func TestActivityLogRepositoryUpdateProjectIDsNullable(t *testing.T) {
	db := newPostgresDB(t)
	repo := NewActivityLogRepository(db, zap.NewNop())

	projectIDs := []uint{7, 8}
	created, err := repo.CreateActivityLogs(context.Background(), CreateActivityLogParams{
		SessionID:  "sess-nullable",
		Method:     "POST",
		APIPart:    "/nullable",
		APIStatus:  "SUCCESS",
		APIAction:  "WRITE",
		ProjectIDs: &projectIDs,
	})
	if err != nil {
		t.Fatalf("create record: %v", err)
	}
	if created == nil {
		t.Fatalf("expected created record")
	}

	var nilProjects []uint
	updated, err := repo.UpdateActivityLogSessionID(context.Background(), UpdateActivityLogSessionParams{
		SessionID:  "sess-nullable",
		ProjectIDs: &nilProjects,
	})
	if err != nil {
		t.Fatalf("update nullable project ids: %v", err)
	}
	if updated.ProjectIDs != nil {
		t.Fatalf("expected project ids to be NULL, got %+v", updated.ProjectIDs)
	}
}

func strPtr(s string) *string {
	return &s
}
