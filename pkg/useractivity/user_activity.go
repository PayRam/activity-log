package useractivity

import (
	"context"
	"fmt"
	"time"

	"github.com/PayRam/user-activity-go/internal/models"
	"github.com/PayRam/user-activity-go/internal/repositories"
	"github.com/PayRam/user-activity-go/internal/services"
	internalutils "github.com/PayRam/user-activity-go/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config configures the user activity client.
type Config struct {
	DB          *gorm.DB
	Logger      *zap.Logger
	TablePrefix string
	TableName   string

	AccessResolver           AccessResolver
	ConfigProvider           ConfigProvider
	MemberResolver           MemberResolver
	ExternalPlatformResolver ExternalPlatformResolver
}

// Client provides Create/Update APIs for user activity records.
type Client struct {
	db     *gorm.DB
	svc    services.UserActivityService
	logger *zap.Logger

	accessResolver           AccessResolver
	configProvider           ConfigProvider
	memberResolver           MemberResolver
	externalPlatformResolver ExternalPlatformResolver
}

// New creates a new user activity client.
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
		models.SetUserActivityTableName(cfg.TableName)
	}

	repo := repositories.NewUserActivityRepository(cfg.DB, logger)
	svc := services.NewUserActivityServiceImpl(repo, logger)

	return &Client{
		db:                       cfg.DB,
		svc:                      svc,
		logger:                   logger,
		accessResolver:           cfg.AccessResolver,
		configProvider:           cfg.ConfigProvider,
		memberResolver:           cfg.MemberResolver,
		externalPlatformResolver: cfg.ExternalPlatformResolver,
	}, nil
}

// AutoMigrate creates or updates the user_activities table schema.
func (c *Client) AutoMigrate(ctx context.Context) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("client is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.db.WithContext(ctx).AutoMigrate(&models.UserActivity{})
}

// Create inserts a new user activity record.
func (c *Client) Create(ctx context.Context, req CreateRequest) (*Activity, error) {
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

	activity := &models.UserActivity{
		MemberID:            req.MemberID,
		SessionID:           req.SessionID,
		ExternalPlatformIDs: models.UintSlice(req.ExternalPlatformIDs),
		Method:              req.Method,
		APIPart:             req.Endpoint,
		APIStatus:           req.APIStatus,
		StatusCode:          req.StatusCode,
		Description:         req.Description,
		IPAddress:           req.IPAddress,
		UserAgent:           req.UserAgent,
		Referer:             req.Referer,
		APIAction:           req.APIAction,
		APIErrorMsg:         req.APIErrorMsg,
		RequestBody:         req.RequestBody,
		ResponseBody:        req.ResponseBody,
		Metadata:            req.Metadata,
		Role:                req.Role,
		EventCategory:       req.EventCategory,
		EventName:           req.EventName,
		Country:             req.Country,
		CountryCode:         req.CountryCode,
		Region:              req.Region,
		City:                req.City,
		Timezone:            req.Timezone,
		Latitude:            req.Latitude,
		Longitude:           req.Longitude,
	}

	created, err := c.svc.Create(ctx, activity)
	if err != nil {
		return nil, err
	}
	return toActivity(created), nil
}

// Update updates an existing activity record by session ID.
func (c *Client) Update(ctx context.Context, req UpdateRequest) (*Activity, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	if req.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	activity := &models.UserActivity{
		SessionID:     req.SessionID,
		MemberID:      req.MemberID,
		StatusCode:    req.StatusCode,
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

	if req.ExternalPlatformIDs != nil {
		activity.ExternalPlatformIDs = models.UintSlice(req.ExternalPlatformIDs)
	}
	if req.Method != nil {
		activity.Method = *req.Method
	}
	if req.Endpoint != nil {
		activity.APIPart = *req.Endpoint
	}
	if req.APIStatus != nil {
		activity.APIStatus = *req.APIStatus
	}
	if req.APIAction != nil {
		activity.APIAction = *req.APIAction
	}

	updated, err := c.svc.UpdateBySessionID(ctx, activity)
	if err != nil {
		return nil, err
	}
	return toActivity(updated), nil
}

// Get retrieves user activities with filtering and access controls.
func (c *Client) Get(ctx context.Context, memberID uint, req GetRequest) (GetResponse, error) {
	if c == nil || c.svc == nil {
		return GetResponse{}, fmt.Errorf("client is not initialized")
	}

	if len(req.ExternalPlatformIDs) > 0 && req.ProjectFilter != nil {
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

	filter := repositories.UserActivityFilters{
		IDS:                 req.IDS,
		APIStatuses:         req.APIStatuses,
		Methods:             req.Methods,
		EventNames:          req.EventNames,
		EventCategories:     req.EventCategories,
		Search:              req.Search,
		StatusCode:          req.StatusCode,
		IPAddresses:         req.IPAddresses,
		Countries:           req.Countries,
		Roles:               req.Roles,
		MemberIDs:           req.MemberIDs,
		SessionID:           req.SessionID,
		ExternalPlatformIDs: req.ExternalPlatformIDs,
		ProjectFilter:       req.ProjectFilter,
		Limit:               req.PaginationConditions.Limit,
		Offset:              req.PaginationConditions.Offset,
		StartDate:           req.PaginationConditions.StartDate,
		EndDate:             req.PaginationConditions.EndDate,
		SortBy:              req.PaginationConditions.SortBy,
		Order:               req.PaginationConditions.Order,
		GreaterThanID:       req.PaginationConditions.GreaterThanID,
		LessThanID:          req.PaginationConditions.LessThanID,
		CreatedAfter:        req.PaginationConditions.CreatedAfter,
		CreatedBefore:       req.PaginationConditions.CreatedBefore,
		UpdatedAfter:        req.PaginationConditions.UpdatedAfter,
		UpdatedBefore:       req.PaginationConditions.UpdatedBefore,
	}

	limit := DefaultGetLimit
	if req.Export {
		if c.configProvider != nil {
			if exportLimit, ok, err := c.configProvider.GetInt(ctx, ConfigKeyUserActivityExportLimit); err == nil && ok && exportLimit > 0 {
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

	activities, total, err := c.svc.Get(ctx, filter)
	if err != nil {
		return GetResponse{}, err
	}

	return c.mapActivities(ctx, activities, total)
}

// GetEventCategories retrieves distinct event categories.
func (c *Client) GetEventCategories(ctx context.Context) ([]string, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	return c.svc.GetEventCategories(ctx)
}

func toActivity(model *models.UserActivity) *Activity {
	if model == nil {
		return nil
	}

	return &Activity{
		ID:                  model.ID,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		MemberID:            model.MemberID,
		SessionID:           model.SessionID,
		ExternalPlatformIDs: []uint(model.ExternalPlatformIDs),
		Method:              model.Method,
		APIPart:             model.APIPart,
		APIStatus:           model.APIStatus,
		StatusCode:          model.StatusCode,
		Description:         model.Description,
		IPAddress:           model.IPAddress,
		UserAgent:           model.UserAgent,
		Referer:             model.Referer,
		APIAction:           model.APIAction,
		APIErrorMsg:         model.APIErrorMsg,
		RequestBody:         model.RequestBody,
		ResponseBody:        model.ResponseBody,
		Metadata:            model.Metadata,
		Role:                model.Role,
		EventCategory:       model.EventCategory,
		EventName:           model.EventName,
		Country:             model.Country,
		CountryCode:         model.CountryCode,
		Region:              model.Region,
		City:                model.City,
		Timezone:            model.Timezone,
		Latitude:            model.Latitude,
		Longitude:           model.Longitude,
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

	if len(req.ExternalPlatformIDs) > 0 {
		for _, id := range req.ExternalPlatformIDs {
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
				req.ExternalPlatformIDs = []uint{0}
			} else {
				req.ExternalPlatformIDs = allowed
			}
			req.ProjectFilter = nil
			return nil
		default:
			return ErrUnauthorized
		}
	}

	if len(allowed) == 0 {
		req.ExternalPlatformIDs = []uint{0}
		return nil
	}
	req.ExternalPlatformIDs = allowed
	return nil
}

func (c *Client) mapActivities(ctx context.Context, modelsList []models.UserActivity, total int64) (GetResponse, error) {
	activities := make([]Activity, 0, len(modelsList))

	memberIDs := internalutils.CollectMemberIDs(modelsList)
	externalPlatformIDs := internalutils.CollectExternalPlatformIDs(modelsList)

	memberInfoMap := map[uint]MemberInfo{}
	if c.memberResolver != nil && len(memberIDs) > 0 {
		info, err := c.memberResolver.GetByIDs(ctx, memberIDs)
		if err != nil {
			return GetResponse{}, err
		}
		memberInfoMap = info
	}

	platformInfoMap := map[uint]ExternalPlatformInfo{}
	if c.externalPlatformResolver != nil && len(externalPlatformIDs) > 0 {
		info, err := c.externalPlatformResolver.GetByIDs(ctx, externalPlatformIDs)
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

		if len(activity.ExternalPlatformIDs) > 0 {
			platforms := make([]ExternalPlatformInfo, 0, len(activity.ExternalPlatformIDs))
			for _, id := range activity.ExternalPlatformIDs {
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
