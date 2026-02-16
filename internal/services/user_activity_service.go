package services

import (
	"context"

	"github.com/PayRam/user-activity-go/internal/models"
	"github.com/PayRam/user-activity-go/internal/repositories"
)

// UserActivityService provides activity persistence behavior.
// UserActivityRepository defines storage operations for user activity.
type UserActivityService interface {
	Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	Get(ctx context.Context, filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
