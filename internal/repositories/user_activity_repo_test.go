package repositories

import (
	"context"
	"testing"

	"github.com/PayRam/user-activity-go/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.UserActivity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestUserActivityRepositoryCreate(t *testing.T) {
	db := newSQLiteDB(t)
	repo := NewUserActivityRepository(db, zap.NewNop())

	if _, err := repo.Create(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil activity")
	}

	activity := &models.UserActivity{
		SessionID: "sess-1",
		Method:    "GET",
		APIPart:   "/ping",
		APIStatus: "SUCCESS",
		APIAction: "READ",
	}
	created, err := repo.Create(context.Background(), activity)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created ID to be set")
	}
}

func TestUserActivityRepositoryListAndCategories(t *testing.T) {
	db := newSQLiteDB(t)
	repo := NewUserActivityRepository(db, zap.NewNop())

	records := []models.UserActivity{
		{SessionID: "s1", Method: "GET", APIPart: "/a", APIStatus: "SUCCESS", APIAction: "READ", EventCategory: strPtr("AUTH")},
		{SessionID: "s2", Method: "POST", APIPart: "/b", APIStatus: "SUCCESS", APIAction: "WRITE", EventCategory: strPtr("PAYMENT")},
		{SessionID: "s3", Method: "GET", APIPart: "/c", APIStatus: "SUCCESS", APIAction: "READ", EventCategory: strPtr("AUTH")},
	}
	for i := range records {
		if _, err := repo.Create(context.Background(), &records[i]); err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	filter := UserActivityFilters{Methods: []string{"GET"}}
	list, total, err := repo.GetUserActivities(context.Background(), filter)
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

func TestUserActivityRepositoryUpdateValidation(t *testing.T) {
	db := newSQLiteDB(t)
	repo := NewUserActivityRepository(db, zap.NewNop())

	if _, err := repo.UpdateBySessionID(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil activity")
	}

	activity := &models.UserActivity{}
	if _, err := repo.UpdateBySessionID(context.Background(), activity); err == nil {
		t.Fatalf("expected error for empty session_id")
	}
}

func TestUserActivityRepositoryUpdateDryRun(t *testing.T) {
	db := newSQLiteDB(t).Session(&gorm.Session{DryRun: true})
	repo := NewUserActivityRepository(db, zap.NewNop())

	activity := &models.UserActivity{
		SessionID: "sess-1",
		APIStatus: "SUCCESS",
		APIAction: "READ",
		Method:    "GET",
		APIPart:   "/ping",
	}
	if _, err := repo.UpdateBySessionID(context.Background(), activity); err != nil {
		t.Fatalf("expected no error in dry run, got %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
