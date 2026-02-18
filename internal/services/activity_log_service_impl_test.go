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
	if f.createErr != nil {
		return nil, f.createErr
	}
	return activity, nil
}

func (f *fakeRepo) UpdateActivityLogSessionID(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	f.updateCalled = true
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return activity, nil
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

	if _, err := svc.CreateActivityLogs(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil activity")
	}
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil activity")
	}
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), &models.ActivityLog{}); err == nil {
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

	act := &models.ActivityLog{SessionID: "s1"}
	if _, err := svc.CreateActivityLogs(context.Background(), act); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected create to be called")
	}

	if _, err := svc.UpdateActivityLogSessionID(context.Background(), act); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if !repo.updateCalled {
		t.Fatalf("expected update to be called")
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

	if _, err := svc.CreateActivityLogs(context.Background(), &models.ActivityLog{}); err == nil {
		t.Fatalf("expected create error")
	}
	if _, err := svc.UpdateActivityLogSessionID(context.Background(), &models.ActivityLog{}); err == nil {
		t.Fatalf("expected update error")
	}
	if _, _, err := svc.GetActivityLogs(context.Background(), repositories.ActivityLogFilters{}); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := svc.GetEventCategories(context.Background()); err == nil {
		t.Fatalf("expected categories error")
	}
}
