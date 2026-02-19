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

	return s.repo.CreateActivityLogs(ctx, activity)
}

// UpdateActivityLogSessionID updates an activity record by session ID.
func (s *ActivityLogServiceImpl) UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	if params.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if params.StatusCode != nil && (*params.StatusCode < 100 || *params.StatusCode > 599) {
		return nil, fmt.Errorf("status_code must be a valid HTTP status code")
	}

	fields := &repositories.ActivityLogUpdateFields{
		MemberID:      params.MemberID,
		Method:        params.Method,
		APIPart:       params.APIPart,
		APIAction:     params.APIAction,
		APIStatus:     params.APIStatus,
		StatusCode:    params.StatusCode,
		Description:   params.Description,
		APIErrorMsg:   params.APIErrorMsg,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
		Referer:       params.Referer,
		ResponseBody:  params.ResponseBody,
		Metadata:      params.Metadata,
		RequestBody:   params.RequestBody,
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
		projectIDs := models.UintSlice(*params.ProjectIDs)
		fields.ProjectIDs = &projectIDs
	}
	if !fields.HasValues() {
		fields = nil
	}

	return s.repo.UpdateActivityLogSessionID(ctx, &repositories.UpdateActivityLogSessionModel{
		SessionID: params.SessionID,
		Fields:    fields,
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
