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
func (r *ActivityLogRepositoryImpl) CreateActivityLogs(ctx context.Context, params CreateActivityLogParams) (*models.ActivityLog, error) {
	activity := &models.ActivityLog{
		MemberID:      params.MemberID,
		SessionID:     params.SessionID,
		Method:        params.Method,
		APIPart:       params.APIPart,
		APIStatus:     params.APIStatus,
		StatusCode:    params.StatusCode,
		Description:   params.Description,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
		Referer:       params.Referer,
		APIAction:     params.APIAction,
		APIErrorMsg:   params.APIErrorMsg,
		RequestBody:   params.RequestBody,
		ResponseBody:  params.ResponseBody,
		Metadata:      params.Metadata,
		Role:          params.Role,
		EventCategory: params.EventCategory,
		EventName:     params.EventName,
		Country:       params.Country,
		CountryCode:   params.CountryCode,
		Region:        params.Region,
		City:          params.City,
		Timezone:      params.Timezone,
		Latitude:      params.Latitude,
		Longitude:     params.Longitude,
	}
	if params.ProjectIDs != nil {
		activity.ProjectIDs = models.UintSlice(*params.ProjectIDs)
	}

	if result := r.db.WithContext(ctx).Create(activity); result.Error != nil {
		if r.logger != nil {
			r.logger.Error("Failed to create activity log", zap.Error(result.Error))
		}
		return nil, fmt.Errorf("failed to create activity: %w", result.Error)
	}
	return activity, nil
}

// UpdateActivityLogSessionID updates an activity record by session ID with row locking.
func (r *ActivityLogRepositoryImpl) UpdateActivityLogSessionID(ctx context.Context, params UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	var activity models.ActivityLog

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_id = ?", params.SessionID).
			First(&activity).Error; err != nil {
			if r.logger != nil {
				r.logger.Error("Failed to lock activity log", zap.String("session_id", params.SessionID), zap.Error(err))
			}
			return fmt.Errorf("failed to lock activity with session_id %s: %w", params.SessionID, err)
		}

		updates := make(map[string]interface{})
		if params.ProjectIDs != nil {
			updates["project_ids"] = models.UintSlice(*params.ProjectIDs)
		}
		if params.MemberID != nil {
			updates["member_id"] = *params.MemberID
		}
		if params.Method != nil {
			updates["method"] = *params.Method
		}
		if params.APIPart != nil {
			updates["api_part"] = *params.APIPart
		}
		if params.APIAction != nil {
			updates["api_action"] = *params.APIAction
		}
		if params.APIStatus != nil {
			updates["api_status"] = *params.APIStatus
		}
		if params.StatusCode != nil {
			updates["status_code"] = *params.StatusCode
		}
		if params.Description != nil {
			updates["description"] = *params.Description
		}
		if params.APIErrorMsg != nil {
			updates["api_error_msg"] = *params.APIErrorMsg
		}
		if params.IPAddress != nil {
			updates["ip_address"] = *params.IPAddress
		}
		if params.UserAgent != nil {
			updates["user_agent"] = *params.UserAgent
		}
		if params.Referer != nil {
			updates["referer"] = *params.Referer
		}
		if params.ResponseBody != nil {
			updates["response_body"] = *params.ResponseBody
		}
		if params.Metadata != nil {
			updates["metadata"] = *params.Metadata
		}
		if params.RequestBody != nil {
			updates["request_body"] = *params.RequestBody
		}
		if params.Role != nil {
			updates["role"] = *params.Role
		}
		if params.EventCategory != nil {
			updates["event_category"] = *params.EventCategory
		}
		if params.EventName != nil {
			updates["event_name"] = *params.EventName
		}
		if params.Country != nil {
			updates["country"] = *params.Country
		}
		if params.CountryCode != nil {
			updates["country_code"] = *params.CountryCode
		}
		if params.Region != nil {
			updates["region"] = *params.Region
		}
		if params.City != nil {
			updates["city"] = *params.City
		}
		if params.Timezone != nil {
			updates["timezone"] = *params.Timezone
		}
		if params.Latitude != nil {
			updates["latitude"] = *params.Latitude
		}
		if params.Longitude != nil {
			updates["longitude"] = *params.Longitude
		}

		if len(updates) > 0 {
			if err := tx.Model(&models.ActivityLog{}).
				Where("session_id = ?", params.SessionID).
				Updates(updates).Error; err != nil {
				if r.logger != nil {
					r.logger.Error("Failed to update activity log", zap.String("session_id", params.SessionID), zap.Error(err))
				}
				return fmt.Errorf("failed to update activity with session_id %s: %w", params.SessionID, err)
			}
		}

		return tx.Where("session_id = ?", params.SessionID).First(&activity).Error
	})

	if err != nil {
		if r.logger != nil {
			r.logger.Error("UpdateActivityLogSessionID failed", zap.String("session_id", params.SessionID), zap.Error(err))
		}
		return nil, err
	}

	return &activity, nil
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
