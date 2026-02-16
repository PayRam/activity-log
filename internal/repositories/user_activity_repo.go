package repositories

import (
	"context"

	"github.com/PayRam/user-activity-go/internal/models"
)

// UserActivityRepository defines storage operations for user activity.
type UserActivityRepository interface {
	Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error)
	GetUserActivities(ctx context.Context, filter UserActivityFilters) ([]models.UserActivity, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
