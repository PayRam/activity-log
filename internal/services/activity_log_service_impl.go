package services

import (
	"context"
	"fmt"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"go.uber.org/zap"
)

// ActivityLogServiceImpl provides activity persistence behavior.
type ActivityLogServiceImpl struct {
	repo   repositories.ActivityLogRepository
	logger *zap.Logger
}

// NewActivityLogServiceImpl creates a new service instance.
func NewActivityLogServiceImpl(repo repositories.ActivityLogRepository, logger *zap.Logger) *ActivityLogServiceImpl {
	return &ActivityLogServiceImpl{repo: repo, logger: logger}
}

// Create persists a new activity record.
func (s *ActivityLogServiceImpl) Create(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	return s.repo.Create(ctx, activity)
}

// UpdateBySessionID updates an activity record by session ID.
func (s *ActivityLogServiceImpl) UpdateBySessionID(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	return s.repo.UpdateBySessionID(ctx, activity)
}

// Get retrieves activities with filtering.
func (s *ActivityLogServiceImpl) Get(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	return s.repo.GetActivityLogs(ctx, filter)
}

// GetEventCategories retrieves distinct event categories.
func (s *ActivityLogServiceImpl) GetEventCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetEventCategories(ctx)
}
