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
func (s *ActivityLogServiceImpl) CreateActivityLogs(ctx context.Context, params repositories.CreateActivityLogParams) (*models.ActivityLog, error) {
	if params.StatusCode != nil && (*params.StatusCode < 100 || *params.StatusCode > 599) {
		return nil, fmt.Errorf("status_code must be a valid HTTP status code")
	}
	return s.repo.CreateActivityLogs(ctx, params)
}

// UpdateActivityLogSessionID updates an activity record by session ID.
func (s *ActivityLogServiceImpl) UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	if params.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if params.StatusCode != nil && (*params.StatusCode < 100 || *params.StatusCode > 599) {
		return nil, fmt.Errorf("status_code must be a valid HTTP status code")
	}
	return s.repo.UpdateActivityLogSessionID(ctx, params)
}

// Get retrieves activities with filtering.
func (s *ActivityLogServiceImpl) GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	return s.repo.GetActivityLogs(ctx, filter)
}

// GetEventCategories retrieves distinct event categories.
func (s *ActivityLogServiceImpl) GetEventCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetEventCategories(ctx)
}
