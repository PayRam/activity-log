package useractivity

import (
	"context"
	"fmt"
	"time"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"github.com/PayRam/activity-log/internal/services"
	internalutils "github.com/PayRam/activity-log/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config configures the activity log client.
type Config struct {
	DB               *gorm.DB
	Logger           *zap.Logger
	TablePrefix      string
	TableName        string
	EventDeriver     EventDeriver
	EventInfoDeriver EventInfoDeriver

	AccessResolver           AccessResolver
	ConfigProvider           ConfigProvider
	MemberResolver           MemberResolver
	ExternalPlatformResolver ExternalPlatformResolver
}

// Client provides Create/Update APIs for activity log records.
type Client struct {
	db               *gorm.DB
	svc              services.ActivityLogService
	logger           *zap.Logger
	eventDeriver     EventDeriver
	eventInfoDeriver EventInfoDeriver

	accessResolver           AccessResolver
	configProvider           ConfigProvider
	memberResolver           MemberResolver
	externalPlatformResolver ExternalPlatformResolver
}

// New creates a new activity log client.
func New(cfg Config) (*Client, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("db is required")
	}

	logger := cfg.Logger
	if logger == nil {
		l, _ := zap.NewProduction()
		logger = l
	}

	if cfg.TablePrefix != "" {
		models.SetTablePrefix(cfg.TablePrefix)
	}
	if cfg.TableName != "" {
		models.SetActivityLogTableName(cfg.TableName)
	}

	repo := repositories.NewActivityLogRepository(cfg.DB, logger)
	svc := services.NewActivityLogServiceImpl(repo, logger)

	return &Client{
		db:                       cfg.DB,
		svc:                      svc,
		logger:                   logger,
		eventDeriver:             cfg.EventDeriver,
		eventInfoDeriver:         cfg.EventInfoDeriver,
		accessResolver:           cfg.AccessResolver,
		configProvider:           cfg.ConfigProvider,
		memberResolver:           cfg.MemberResolver,
		externalPlatformResolver: cfg.ExternalPlatformResolver,
	}, nil
}

// AutoMigrate creates or updates the activity_logs table schema.
func (c *Client) AutoMigrate(ctx context.Context) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("client is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.db.WithContext(ctx).AutoMigrate(&models.ActivityLog{})
}

// Create inserts a new activity log record.
func (c *Client) CreateActivityLogs(ctx context.Context, req CreateRequest) (*Activity, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("method is required")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if req.APIAction == "" {
		return nil, fmt.Errorf("api_action is required")
	}
	if req.APIStatus == "" {
		return nil, fmt.Errorf("api_status is required")
	}

	applyEventFallback(&req, c.eventDeriver, c.eventInfoDeriver)

	params := repositories.CreateActivityLogParams{
		MemberID:      req.MemberID,
		SessionID:     req.SessionID,
		ProjectIDs:    uintSlicePtr(req.ProjectIDs),
		Method:        req.Method,
		APIPart:       req.Endpoint,
		APIStatus:     string(req.APIStatus),
		StatusCode:    statusCodePtrToInt(req.StatusCode),
		Description:   req.Description,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Referer:       req.Referer,
		APIAction:     req.APIAction,
		APIErrorMsg:   req.APIErrorMsg,
		RequestBody:   req.RequestBody,
		ResponseBody:  req.ResponseBody,
		Metadata:      req.Metadata,
		Role:          req.Role,
		EventCategory: req.EventCategory,
		EventName:     req.EventName,
		Country:       req.Country,
		CountryCode:   req.CountryCode,
		Region:        req.Region,
		City:          req.City,
		Timezone:      req.Timezone,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
	}

	created, err := c.svc.CreateActivityLogs(ctx, params)
	if err != nil {
		return nil, err
	}
	return toActivity(created), nil
}

// Update updates an existing activity record by session ID.
func (c *Client) UpdateActivityLogSessionID(ctx context.Context, req UpdateRequest) (*Activity, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	applyUpdateEventFallback(&req, c.eventDeriver, c.eventInfoDeriver)

	params := repositories.UpdateActivityLogSessionParams{
		SessionID:     req.SessionID,
		ProjectIDs:    cloneUintSlicePtr(req.ProjectIDs),
		MemberID:      req.MemberID,
		Method:        req.Method,
		APIPart:       req.Endpoint,
		APIAction:     req.APIAction,
		StatusCode:    statusCodePtrToInt(req.StatusCode),
		Description:   req.Description,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		Referer:       req.Referer,
		APIErrorMsg:   req.APIErrorMsg,
		Metadata:      req.Metadata,
		RequestBody:   req.RequestBody,
		ResponseBody:  req.ResponseBody,
		Role:          req.Role,
		EventCategory: req.EventCategory,
		EventName:     req.EventName,
		Country:       req.Country,
		CountryCode:   req.CountryCode,
		Region:        req.Region,
		City:          req.City,
		Timezone:      req.Timezone,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
	}
	if req.APIStatus != nil {
		status := string(*req.APIStatus)
		params.APIStatus = &status
	}

	updated, err := c.svc.UpdateActivityLogSessionID(ctx, params)
	if err != nil {
		return nil, err
	}
	return toActivity(updated), nil
}

// Get retrieves activity logs with filtering and access controls.
func (c *Client) GetActivityLogs(ctx context.Context, memberID uint, req GetRequest) (GetResponse, error) {
	if c == nil || c.svc == nil {
		return GetResponse{}, fmt.Errorf("client is not initialized")
	}

	if len(req.ProjectIDs) > 0 && req.ProjectFilter != nil {
		return GetResponse{}, ErrBadRequest
	}

	if memberID > 0 && c.accessResolver != nil {
		access, err := c.accessResolver.Resolve(ctx, memberID)
		if err != nil {
			return GetResponse{}, err
		}
		if err := applyAccessScope(&req, access); err != nil {
			return GetResponse{}, err
		}
	}

	filter := repositories.ActivityLogFilters{
		IDS:             req.IDS,
		APIStatuses:     apiStatusesToStrings(req.APIStatuses),
		Methods:         req.Methods,
		EventNames:      req.EventNames,
		EventCategories: req.EventCategories,
		Search:          req.Search,
		StatusCodes:     statusCodesToInts(req.StatusCodes),
		IPAddresses:     req.IPAddresses,
		Countries:       req.Countries,
		Roles:           req.Roles,
		MemberIDs:       req.MemberIDs,
		SessionIDs:      req.SessionIDs,
		ProjectIDs:      req.ProjectIDs,
		ProjectFilter:   req.ProjectFilter,
		Limit:           req.PaginationConditions.Limit,
		Offset:          req.PaginationConditions.Offset,
		StartDate:       req.PaginationConditions.StartDate,
		EndDate:         req.PaginationConditions.EndDate,
		SortBy:          req.PaginationConditions.SortBy,
		Order:           req.PaginationConditions.Order,
		GreaterThanID:   req.PaginationConditions.GreaterThanID,
		LessThanID:      req.PaginationConditions.LessThanID,
		CreatedAfter:    req.PaginationConditions.CreatedAfter,
		CreatedBefore:   req.PaginationConditions.CreatedBefore,
		UpdatedAfter:    req.PaginationConditions.UpdatedAfter,
		UpdatedBefore:   req.PaginationConditions.UpdatedBefore,
	}

	limit := DefaultGetLimit
	if req.Export {
		if c.configProvider != nil {
			if exportLimit, ok, err := c.configProvider.GetInt(ctx, ConfigKeyActivityLogExportLimit); err == nil && ok && exportLimit > 0 {
				limit = exportLimit
			}
		}
	}

	if filter.Limit == nil || *filter.Limit <= 0 {
		filter.Limit = &limit
	} else if *filter.Limit > limit {
		filter.Limit = &limit
	}

	if filter.StartDate != nil {
		if filter.EndDate == nil {
			endDate := time.Now()
			filter.EndDate = &endDate
		} else if filter.EndDate.After(time.Now()) {
			endDate := time.Now()
			filter.EndDate = &endDate
		}
	}

	activities, total, err := c.svc.GetActivityLogs(ctx, filter)
	if err != nil {
		return GetResponse{}, err
	}

	return c.mapActivities(ctx, activities, total)
}

func applyEventFallback(req *CreateRequest, eventDeriver EventDeriver, eventInfoDeriver EventInfoDeriver) {
	if req == nil {
		return
	}
	if req.EventCategory != nil && req.EventName != nil && req.Description != nil {
		return
	}

	info := deriveEventInfo(EventDeriverInput{
		Endpoint:    req.Endpoint,
		Method:      req.Method,
		RequestBody: req.RequestBody,
		StatusCode:  req.StatusCode,
		APIStatus:   req.APIStatus,
	}, eventDeriver, eventInfoDeriver)

	if req.EventCategory == nil {
		if info.EventCategory == "" {
			return
		}
		category := info.EventCategory
		req.EventCategory = &category
	}
	if req.EventName == nil {
		if info.EventName == "" {
			return
		}
		name := info.EventName
		req.EventName = &name
	}
	if req.Description == nil && info.Description != "" {
		description := info.Description
		req.Description = &description
	}
}

func applyUpdateEventFallback(req *UpdateRequest, eventDeriver EventDeriver, eventInfoDeriver EventInfoDeriver) {
	if req == nil {
		return
	}
	if req.EventCategory != nil && req.EventName != nil && req.Description != nil {
		return
	}

	endpoint := ""
	if req.Endpoint != nil {
		endpoint = *req.Endpoint
	}
	method := ""
	if req.Method != nil {
		method = *req.Method
	}
	apiStatus := APIStatus("")
	if req.APIStatus != nil {
		apiStatus = *req.APIStatus
	}

	if endpoint == "" && method == "" {
		return
	}

	info := deriveEventInfo(EventDeriverInput{
		Endpoint:    endpoint,
		Method:      method,
		RequestBody: req.RequestBody,
		StatusCode:  req.StatusCode,
		APIStatus:   apiStatus,
	}, eventDeriver, eventInfoDeriver)

	if req.EventCategory == nil && info.EventCategory != "" {
		category := info.EventCategory
		req.EventCategory = &category
	}
	if req.EventName == nil && info.EventName != "" {
		name := info.EventName
		req.EventName = &name
	}
	if req.Description == nil && info.Description != "" {
		description := info.Description
		req.Description = &description
	}
}

// GetEventCategories retrieves distinct event categories.
func (c *Client) GetEventCategories(ctx context.Context) ([]string, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	return c.svc.GetEventCategories(ctx)
}

func toActivity(model *models.ActivityLog) *Activity {
	if model == nil {
		return nil
	}

	return &Activity{
		ID:            model.ID,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
		MemberID:      model.MemberID,
		SessionID:     model.SessionID,
		ProjectIDs:    []uint(model.ProjectIDs),
		Method:        model.Method,
		APIPart:       model.APIPart,
		APIStatus:     APIStatus(model.APIStatus),
		StatusCode:    statusCodePtrFromInt(model.StatusCode),
		Description:   model.Description,
		IPAddress:     model.IPAddress,
		UserAgent:     model.UserAgent,
		Referer:       model.Referer,
		APIAction:     model.APIAction,
		APIErrorMsg:   model.APIErrorMsg,
		RequestBody:   model.RequestBody,
		ResponseBody:  model.ResponseBody,
		Metadata:      model.Metadata,
		Role:          model.Role,
		EventCategory: model.EventCategory,
		EventName:     model.EventName,
		Country:       model.Country,
		CountryCode:   model.CountryCode,
		Region:        model.Region,
		City:          model.City,
		Timezone:      model.Timezone,
		Latitude:      model.Latitude,
		Longitude:     model.Longitude,
	}
}

func applyAccessScope(req *GetRequest, access *AccessContext) error {
	if req == nil || access == nil || access.IsAdmin {
		return nil
	}

	allowed := access.AllowedProjectIDs
	allowedSet := make(map[uint]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}

	if len(req.ProjectIDs) > 0 {
		for _, id := range req.ProjectIDs {
			if _, ok := allowedSet[id]; !ok {
				return ErrUnauthorized
			}
		}
		return nil
	}

	if req.ProjectFilter != nil {
		switch *req.ProjectFilter {
		case "NO_IDS":
			return ErrUnauthorized
		case "ALL":
			if len(allowed) == 0 {
				req.ProjectIDs = []uint{0}
			} else {
				req.ProjectIDs = allowed
			}
			req.ProjectFilter = nil
			return nil
		default:
			return ErrUnauthorized
		}
	}

	if len(allowed) == 0 {
		req.ProjectIDs = []uint{0}
		return nil
	}
	req.ProjectIDs = allowed
	return nil
}

func (c *Client) mapActivities(ctx context.Context, modelsList []models.ActivityLog, total int64) (GetResponse, error) {
	activities := make([]Activity, 0, len(modelsList))

	memberIDs := internalutils.CollectMemberIDs(modelsList)
	projectIDs := internalutils.CollectProjectIDs(modelsList)

	memberInfoMap := map[uint]MemberInfo{}
	if c.memberResolver != nil && len(memberIDs) > 0 {
		info, err := c.memberResolver.GetByIDs(ctx, memberIDs)
		if err != nil {
			return GetResponse{}, err
		}
		memberInfoMap = info
	}

	platformInfoMap := map[uint]ExternalPlatformInfo{}
	if c.externalPlatformResolver != nil && len(projectIDs) > 0 {
		info, err := c.externalPlatformResolver.GetByIDs(ctx, projectIDs)
		if err != nil {
			return GetResponse{}, err
		}
		platformInfoMap = info
	}

	for _, model := range modelsList {
		activity := toActivity(&model)
		if activity == nil {
			continue
		}

		if activity.MemberID != nil {
			if info, ok := memberInfoMap[*activity.MemberID]; ok {
				copyInfo := info
				activity.Member = &copyInfo
			}
		}

		if len(activity.ProjectIDs) > 0 {
			platforms := make([]ExternalPlatformInfo, 0, len(activity.ProjectIDs))
			for _, id := range activity.ProjectIDs {
				if info, ok := platformInfoMap[id]; ok {
					platforms = append(platforms, info)
				}
			}
			activity.ExternalPlatforms = platforms
		}

		activities = append(activities, *activity)
	}

	return GetResponse{Activities: activities, TotalCount: total}, nil
}

func statusCodePtrToInt(code *HTTPStatusCode) *int {
	if code == nil {
		return nil
	}
	value := int(*code)
	return &value
}

func statusCodePtrFromInt(code *int) *HTTPStatusCode {
	if code == nil {
		return nil
	}
	value := HTTPStatusCode(*code)
	return &value
}

func statusCodesToInts(codes []HTTPStatusCode) []int {
	if len(codes) == 0 {
		return nil
	}
	values := make([]int, 0, len(codes))
	for _, code := range codes {
		values = append(values, int(code))
	}
	return values
}

func uintSlicePtr(values []uint) *[]uint {
	if values == nil {
		return nil
	}
	out := make([]uint, len(values))
	copy(out, values)
	return &out
}

func cloneUintSlicePtr(values *[]uint) *[]uint {
	if values == nil {
		return nil
	}
	if *values == nil {
		var nilSlice []uint
		return &nilSlice
	}
	out := make([]uint, len(*values))
	copy(out, *values)
	return &out
}

func apiStatusesToStrings(statuses []APIStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if status == "" {
			continue
		}
		values = append(values, string(status))
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
