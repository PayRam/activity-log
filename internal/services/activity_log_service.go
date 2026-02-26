package services

import (
	"context"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
)

// ActivityLogService provides activity persistence behavior.
// It orchestrates create/update/read operations over activity log records.
type ActivityLogService interface {
	CreateActivityLogs(ctx context.Context, params repositories.CreateActivityLogParams) (*models.ActivityLog, error)
	UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error)
	GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
