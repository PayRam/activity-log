package repositories

import (
	"context"

	"github.com/PayRam/activity-log/internal/models"
)

// ActivityLogRepository defines storage operations for activity log.
type ActivityLogRepository interface {
	CreateActivityLogs(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error)
	UpdateActivityLogSessionID(ctx context.Context, update *UpdateActivityLogSessionModel) (*models.ActivityLog, error)
	GetActivityLogs(ctx context.Context, filter ActivityLogFilters) ([]models.ActivityLog, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
