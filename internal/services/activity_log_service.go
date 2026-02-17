package services

import (
	"context"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
)

// ActivityLogService provides activity persistence behavior.
// It orchestrates create/update/read operations over activity log records.
type ActivityLogService interface {
	Create(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error)
	UpdateBySessionID(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error)
	Get(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error)
	GetEventCategories(ctx context.Context) ([]string, error)
}
