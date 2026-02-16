package services

import (
	"context"
	"fmt"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"go.uber.org/zap"
)

// UserActivityServiceImpl provides activity persistence behavior.
type UserActivityServiceImpl struct {
	repo   repositories.UserActivityRepository
	logger *zap.Logger
}

// NewUserActivityServiceImpl creates a new service instance.
func NewUserActivityServiceImpl(repo repositories.UserActivityRepository, logger *zap.Logger) *UserActivityServiceImpl {
	return &UserActivityServiceImpl{repo: repo, logger: logger}
}

// Create persists a new activity record.
func (s *UserActivityServiceImpl) Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	return s.repo.Create(ctx, activity)
}

// UpdateBySessionID updates an activity record by session ID.
func (s *UserActivityServiceImpl) UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	return s.repo.UpdateBySessionID(ctx, activity)
}

// Get retrieves activities with filtering.
func (s *UserActivityServiceImpl) Get(ctx context.Context, filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
	return s.repo.GetUserActivities(ctx, filter)
}

// GetEventCategories retrieves distinct event categories.
func (s *UserActivityServiceImpl) GetEventCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetEventCategories(ctx)
}
