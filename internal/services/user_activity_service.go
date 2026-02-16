package services

import (
	"context"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
)

// UserActivityService provides activity persistence behavior.
// It orchestrates create/update/read operations over user activity records.
type UserActivityService interface {
	Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	Get(ctx context.Context, filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
