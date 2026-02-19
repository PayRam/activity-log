package services

import (
	"context"
	"errors"
	"testing"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"go.uber.org/zap"
)

type fakeRepo struct {
	createCalled bool
	updateCalled bool
	listCalled   bool
	catCalled    bool

	lastCreate *models.ActivityLog
	lastUpdate *models.ActivityLog

	createErr error
	updateErr error
	listErr   error
	catErr    error

	listItems []models.ActivityLog
	total     int64
	cats      []string
}

func (f *fakeRepo) CreateActivityLogs(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	f.createCalled = true
	f.lastCreate = activity
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &models.ActivityLog{SessionID: activity.SessionID}, nil
}

func (f *fakeRepo) UpdateActivityLogSessionID(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	f.updateCalled = true
	f.lastUpdate = activity
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return &models.ActivityLog{SessionID: activity.SessionID}, nil
}

func (f *fakeRepo) GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	f.listCalled = true
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listItems, f.total, nil
}

func (f *fakeRepo) GetEventCategories(ctx context.Context) ([]string, error) {
	f.catCalled = true
	if f.catErr != nil {
		return nil, f.catErr
	}
	return f.cats, nil
}

func TestActivityLogServiceValidation(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewActivityLogServiceImpl(repo, zap.NewNop())

	if _, err := svc.UpdateActivityLogSessionID(context.Background(), repositories.UpdateActivityLogSessionParams{}); err == nil {
		t.Fatalf("expected error for empty session_id")
	}
}

func TestActivityLogServicePassThrough(t *testing.T) {
	repo := &fakeRepo{
		listItems: []models.ActivityLog{{SessionID: "s1"}},
		total:     1,
		cats:      []string{"AUTH"},
	}
	svc := NewActivityLogServiceImpl(repo, zap.NewNop())

	if _, err := svc.CreateActivityLogs(context.Background(), repositories.CreateActivityLogParams{SessionID: "s1"}); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected create to be called")
	}
	if repo.lastCreate == nil || repo.lastCreate.SessionID != "s1" {
		t.Fatalf("expected service to map create params into model")
	}

	method := "PATCH"
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), repositories.UpdateActivityLogSessionParams{SessionID: "s1", Method: &method}); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if !repo.updateCalled {
		t.Fatalf("expected update to be called")
	}
	if repo.lastUpdate == nil || repo.lastUpdate.SessionID != "s1" {
		t.Fatalf("expected update session to be mapped")
	}
	methodValue, ok := repo.lastUpdate.UpdateFields["method"]
	if !ok || methodValue != "PATCH" {
		t.Fatalf("expected service to map update params into update fields")
	}

	activities, total, err := svc.GetActivityLogs(context.Background(), repositories.ActivityLogFilters{})
	if err != nil {
		t.Fatalf("unexpected activities error: %v", err)
	}
	if len(activities) != 1 || total != 1 {
		t.Fatalf("unexpected activities result")
	}

	categories, err := svc.GetEventCategories(context.Background())
	if err != nil {
		t.Fatalf("unexpected categories error: %v", err)
	}
	if len(categories) != 1 || categories[0] != "AUTH" {
		t.Fatalf("unexpected categories result")
	}
}

func TestActivityLogServiceErrors(t *testing.T) {
	repo := &fakeRepo{
		createErr: errors.New("create"),
		updateErr: errors.New("update"),
		listErr:   errors.New("list"),
		catErr:    errors.New("cat"),
	}
	svc := NewActivityLogServiceImpl(repo, zap.NewNop())

	if _, err := svc.CreateActivityLogs(context.Background(), repositories.CreateActivityLogParams{}); err == nil {
		t.Fatalf("expected create error")
	}
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), repositories.UpdateActivityLogSessionParams{SessionID: "s1"}); err == nil {
		t.Fatalf("expected update error")
	}
	if _, _, err := svc.GetActivityLogs(context.Background(), repositories.ActivityLogFilters{}); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := svc.GetEventCategories(context.Background()); err == nil {
		t.Fatalf("expected categories error")
	}
}

func TestActivityLogServiceStatusCodeValidation(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewActivityLogServiceImpl(repo, zap.NewNop())

	invalid := 99
	if _, err := svc.CreateActivityLogs(context.Background(), repositories.CreateActivityLogParams{StatusCode: &invalid}); err == nil {
		t.Fatalf("expected create error for invalid status_code")
	}

	invalid = 700
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), repositories.UpdateActivityLogSessionParams{SessionID: "s1", StatusCode: &invalid}); err == nil {
		t.Fatalf("expected update error for invalid status_code")
	}
}
