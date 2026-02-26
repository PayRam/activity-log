package services

import (
	"context"
	"fmt"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"go.uber.org/zap"
)

// ActivityLogServiceImpl provides activity persistence behavior.
type ActivityLogServiceImpl struct {
	repo   repositories.ActivityLogRepository
	logger *zap.Logger
}

// NewActivityLogServiceImpl creates a new service instance.
func NewActivityLogServiceImpl(repo repositories.ActivityLogRepository, logger *zap.Logger) *ActivityLogServiceImpl {
	return &ActivityLogServiceImpl{repo: repo, logger: logger}
}

// Create persists a new activity record.
func (s *ActivityLogServiceImpl) CreateActivityLogs(ctx context.Context, params repositories.CreateActivityLogParams) (*models.ActivityLog, error) {
	if err := validateHTTPStatusCode(params.StatusCode); err != nil {
		return nil, err
	}

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
		activity.ProjectIDs = models.UintSlice(params.ProjectIDs)
	}

	return s.repo.CreateActivityLogs(ctx, activity)
}

// UpdateActivityLogSessionID updates an activity record by session ID.
func (s *ActivityLogServiceImpl) UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	if params.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if err := validateHTTPStatusCode(params.StatusCode); err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if params.ProjectIDs != nil {
		updates["project_ids"] = models.UintSlice(params.ProjectIDs)
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

	if len(updates) == 0 {
		updates = nil
	}

	return s.repo.UpdateActivityLogSessionID(ctx, repositories.UpdateActivityLogSessionModel{
		SessionID: params.SessionID,
		Updates:   updates,
	})
}

// Get retrieves activities with filtering.
func (s *ActivityLogServiceImpl) GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	return s.repo.GetActivityLogs(ctx, filter)
}

// GetEventCategories retrieves distinct event categories.
func (s *ActivityLogServiceImpl) GetEventCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetEventCategories(ctx)
}

func validateHTTPStatusCode(statusCode *int) error {
	if statusCode == nil {
		return nil
	}
	if *statusCode < 100 || *statusCode > 599 {
		return fmt.Errorf("status_code must be a valid HTTP status code")
	}
	return nil
}
