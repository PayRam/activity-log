package repositories

import (
	"context"
	"fmt"

	"github.com/PayRam/user-activity-go/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserActivityRepositoryGorm implements UserActivityRepository using GORM.
type UserActivityRepositoryGorm struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewUserActivityRepository creates a new GORM-backed repository.
func NewUserActivityRepository(db *gorm.DB, logger *zap.Logger) *UserActivityRepositoryGorm {
	return &UserActivityRepositoryGorm{db: db, logger: logger}
}

// Create inserts a new activity record.
func (r *UserActivityRepositoryGorm) Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	if result := r.db.WithContext(ctx).Create(activity); result.Error != nil {
		if r.logger != nil {
			r.logger.Error("Failed to create user activity", zap.Error(result.Error))
		}
		return nil, fmt.Errorf("failed to create activity: %w", result.Error)
	}
	return activity, nil
}

// UpdateBySessionID updates an activity record by session ID with row locking.
func (r *UserActivityRepositoryGorm) UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	if activity == nil {
		return nil, fmt.Errorf("activity is nil")
	}
	if activity.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock row for update
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_id = ?", activity.SessionID).
			First(&models.UserActivity{}).Error; err != nil {
			if r.logger != nil {
				r.logger.Error("Failed to lock user activity", zap.String("session_id", activity.SessionID), zap.Error(err))
			}
			return fmt.Errorf("failed to lock activity with session_id %s: %w", activity.SessionID, err)
		}

		// Update using struct (zero values omitted by GORM)
		if err := tx.Model(&models.UserActivity{}).
			Where("session_id = ?", activity.SessionID).
			Updates(activity).Error; err != nil {
			if r.logger != nil {
				r.logger.Error("Failed to update user activity", zap.String("session_id", activity.SessionID), zap.Error(err))
			}
			return fmt.Errorf("failed to update activity with session_id %s: %w", activity.SessionID, err)
		}

		// Reload updated record
		return tx.Where("session_id = ?", activity.SessionID).First(activity).Error
	})

	if err != nil {
		if r.logger != nil {
			r.logger.Error("UpdateBySessionID failed", zap.String("session_id", activity.SessionID), zap.Error(err))
		}
		return nil, err
	}

	return activity, nil
}

// GetUserActivities retrieves activities with filtering and pagination.
func (r *UserActivityRepositoryGorm) GetUserActivities(ctx context.Context, filter UserActivityFilters) ([]models.UserActivity, int64, error) {
	var activities []models.UserActivity
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&models.UserActivity{})
	query = ApplyUserActivityGetFilters(query, filter)

	if err := query.Count(&totalCount).Error; err != nil {
		if r.logger != nil {
			r.logger.Error("Failed to count user activities", zap.Error(err))
		}
		return nil, 0, fmt.Errorf("failed to count activities: %w", err)
	}

	query = ApplyUserActivitiesPaginationConditions(query, filter)

	if result := query.Order("created_at DESC").Find(&activities); result.Error != nil {
		if r.logger != nil {
			r.logger.Error("Failed to fetch user activities", zap.Error(result.Error))
		}
		return nil, 0, fmt.Errorf("failed to fetch activities: %w", result.Error)
	}

	return activities, totalCount, nil
}

// GetEventCategories retrieves distinct event categories.
func (r *UserActivityRepositoryGorm) GetEventCategories(ctx context.Context) ([]string, error) {
	var categories []string

	if err := r.db.WithContext(ctx).
		Model(&models.UserActivity{}).
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
