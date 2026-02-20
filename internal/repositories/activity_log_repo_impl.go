package repositories

import (
	"context"
	"fmt"

	"github.com/PayRam/activity-log/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ActivityLogRepositoryImpl implements ActivityLogRepository using GORM.
type ActivityLogRepositoryImpl struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewActivityLogRepository creates a new GORM-backed repository.
func NewActivityLogRepository(db *gorm.DB, logger *zap.Logger) *ActivityLogRepositoryImpl {
	return &ActivityLogRepositoryImpl{db: db, logger: logger}
}

// Create inserts a new activity record.
func (r *ActivityLogRepositoryImpl) CreateActivityLogs(ctx context.Context, activity *models.ActivityLog) (*models.ActivityLog, error) {
	if result := r.db.WithContext(ctx).Create(activity); result.Error != nil {
		if r.logger != nil {
			r.logger.Error("Failed to create activity log", zap.Error(result.Error))
		}
		return nil, fmt.Errorf("failed to create activity: %w", result.Error)
	}
	return activity, nil
}

// UpdateActivityLogSessionID updates an activity record by session ID with row locking.
func (r *ActivityLogRepositoryImpl) UpdateActivityLogSessionID(ctx context.Context, update *UpdateActivityLogSessionModel) (*models.ActivityLog, error) {
	var updated models.ActivityLog

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_id = ?", update.SessionID).
			First(&updated).Error; err != nil {
			if r.logger != nil {
				r.logger.Error("Failed to lock activity log", zap.String("session_id", update.SessionID), zap.Error(err))
			}
			return fmt.Errorf("failed to lock activity with session_id %s: %w", update.SessionID, err)
		}

		if len(update.Updates) > 0 {
			if err := tx.Model(&models.ActivityLog{}).
				Where("session_id = ?", update.SessionID).
				Updates(update.Updates).Error; err != nil {
				if r.logger != nil {
					r.logger.Error("Failed to update activity log", zap.String("session_id", update.SessionID), zap.Error(err))
				}
				return fmt.Errorf("failed to update activity with session_id %s: %w", update.SessionID, err)
			}
		}

		return tx.Where("session_id = ?", update.SessionID).First(&updated).Error
	})

	if err != nil {
		if r.logger != nil {
			r.logger.Error("UpdateActivityLogSessionID failed", zap.String("session_id", update.SessionID), zap.Error(err))
		}
		return nil, err
	}

	return &updated, nil
}

// GetActivityLogs retrieves activities with filtering and pagination.
func (r *ActivityLogRepositoryImpl) GetActivityLogs(ctx context.Context, filter ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	var activities []models.ActivityLog
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.ActivityLog{})
	query = ApplyActivityLogGetFilters(query, filter)

	if err := query.Count(&totalCount).Error; err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to count activity logs", zap.Error(err))
		}
		return nil, 0, fmt.Errorf("failed to count activities: %w", err)
	}

	query = ApplyActivityLogsPaginationConditions(query, filter)
	if filter.Order == nil {
		query = query.Order("created_at DESC")
	}

	if result := query.Find(&activities); result.Error != nil {
		if r.logger != nil {
			r.logger.Error("Failed to fetch activity logs", zap.Error(result.Error))
		}
		return nil, 0, fmt.Errorf("failed to fetch activities: %w", result.Error)
	}

	return activities, totalCount, nil
}

// GetEventCategories retrieves distinct event categories.
func (r *ActivityLogRepositoryImpl) GetEventCategories(ctx context.Context) ([]string, error) {
	var categories []string

	if err := r.db.WithContext(ctx).
		Model(&models.ActivityLog{}).
		Where("event_category IS NOT NULL").
		Distinct("event_category").
		Order("event_category ASC").
		Pluck("event_category", &categories).Error; err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to fetch event categories", zap.Error(err))
		}
		return nil, fmt.Errorf("failed to fetch event categories: %w", err)
	}

	return categories, nil
}
