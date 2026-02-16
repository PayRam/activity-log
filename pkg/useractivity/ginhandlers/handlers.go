package ginhandlers

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PayRam/activity-log/pkg/useractivity"
	"github.com/gin-gonic/gin"
)

// HandlerConfig configures the Gin handlers.
type HandlerConfig struct {
	Client              *useractivity.Client
	MemberIDFromContext func(*gin.Context) (uint, bool)
	RequireMember       bool
	ErrorHandler        func(*gin.Context, error)
}

// Handler implements user activity HTTP handlers.
type Handler struct {
	client              *useractivity.Client
	memberIDFromContext func(*gin.Context) (uint, bool)
	requireMember       bool
	errorHandler        func(*gin.Context, error)
}

// NewHandler creates a new handler.
func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		client:              cfg.Client,
		memberIDFromContext: cfg.MemberIDFromContext,
		requireMember:       cfg.RequireMember,
		errorHandler:        cfg.ErrorHandler,
	}
}

// SetupRoutes registers user activity routes.
func SetupRoutes(router *gin.RouterGroup, handler *Handler, middleware ...gin.HandlerFunc) {
	if handler == nil || router == nil {
		return
	}

	group := router
	if len(middleware) > 0 {
		group = router.Group("", middleware...)
	}

	group.GET("/user-activity", handler.GetUserActivities)
	group.GET("/user-activity/event-categories", handler.GetEventCategories)
	group.GET("/user-activity/export", handler.DownloadUserActivitiesCSV)
}

// GetUserActivities handles GET /user-activity.
func (h *Handler) GetUserActivities(c *gin.Context) {
	memberID, ok := h.getMemberID(c)
	if h.requireMember && !ok {
		h.handleError(c, useractivity.ErrUnauthorized)
		return
	}

	var queryParams useractivity.GetRequest
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		h.handleError(c, useractivity.ErrBadRequest)
		return
	}
	if len(queryParams.ExternalPlatformIDs) > 0 && queryParams.ProjectFilter != nil {
		h.handleError(c, useractivity.ErrBadRequest)
		return
	}

	resp, err := h.client.Get(c.Request.Context(), memberID, queryParams)
	if err != nil {
		h.handleError(c, err)
		return
	}

	sanitized := make([]map[string]interface{}, 0, len(resp.Activities))
	for _, activity := range resp.Activities {
		externalPlatforms := make([]map[string]interface{}, 0, len(activity.ExternalPlatforms))
		for _, ep := range activity.ExternalPlatforms {
			externalPlatforms = append(externalPlatforms, map[string]interface{}{
				"id":        ep.ID,
				"name":      ep.Name,
				"logo_path": ep.LogoPath,
			})
		}

		sanitized = append(sanitized, map[string]interface{}{
			"id":                    activity.ID,
			"session_id":            activity.SessionID,
			"member_id":             activity.MemberID,
			"member":                activity.Member,
			"api_action":            activity.APIAction,
			"api_part":              activity.APIPart,
			"method":                activity.Method,
			"api_status":            activity.APIStatus,
			"description":           activity.Description,
			"ip_address":            activity.IPAddress,
			"user_agent":            activity.UserAgent,
			"referer":               activity.Referer,
			"status_code":           activity.StatusCode,
			"api_error_msg":         activity.APIErrorMsg,
			"metadata":              activity.Metadata,
			"role":                  activity.Role,
			"event_category":        activity.EventCategory,
			"event_name":            activity.EventName,
			"country":               activity.Country,
			"country_code":          activity.CountryCode,
			"region":                activity.Region,
			"city":                  activity.City,
			"timezone":              activity.Timezone,
			"latitude":              activity.Latitude,
			"longitude":             activity.Longitude,
			"external_platform_ids": activity.ExternalPlatformIDs,
			"external_platforms":    externalPlatforms,
			"created_at":            activity.CreatedAt,
			"updated_at":            activity.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_count": resp.TotalCount,
		"data":        sanitized,
	})
}

// DownloadUserActivitiesCSV handles GET /user-activity/export.
func (h *Handler) DownloadUserActivitiesCSV(c *gin.Context) {
	memberID, ok := h.getMemberID(c)
	if h.requireMember && !ok {
		h.handleError(c, useractivity.ErrUnauthorized)
		return
	}

	var queryParams useractivity.GetRequest
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		h.handleError(c, useractivity.ErrBadRequest)
		return
	}
	if len(queryParams.ExternalPlatformIDs) > 0 && queryParams.ProjectFilter != nil {
		h.handleError(c, useractivity.ErrBadRequest)
		return
	}
	queryParams.Export = true

	resp, err := h.client.Get(c.Request.Context(), memberID, queryParams)
	if err != nil {
		h.handleError(c, err)
		return
	}

	var csvBuffer bytes.Buffer
	writer := csv.NewWriter(&csvBuffer)

	header := []string{
		"id",
		"session_id",
		"member_id",
		"member_name",
		"member_email",
		"member_username",
		"api_action",
		"api_part",
		"method",
		"api_status",
		"description",
		"ip_address",
		"user_agent",
		"referer",
		"status_code",
		"api_error_msg",
		"metadata",
		"role",
		"event_category",
		"event_name",
		"country",
		"country_code",
		"region",
		"city",
		"timezone",
		"latitude",
		"longitude",
		"external_platform_ids",
		"external_platform_names",
		"created_at",
		"updated_at",
	}

	if err := writer.Write(header); err != nil {
		h.handleError(c, err)
		return
	}

	for _, activity := range resp.Activities {
		memberName := ""
		memberEmail := ""
		memberUsername := ""
		if activity.Member != nil {
			memberName = activity.Member.Name
			if activity.Member.Email != nil {
				memberEmail = *activity.Member.Email
			}
			if activity.Member.Username != nil {
				memberUsername = *activity.Member.Username
			}
		}

		externalPlatformIDs := formatUintSlice(activity.ExternalPlatformIDs)
		externalPlatformNames := make([]string, 0, len(activity.ExternalPlatforms))
		for _, platform := range activity.ExternalPlatforms {
			externalPlatformNames = append(externalPlatformNames, platform.Name)
		}

		row := []string{
			strconv.FormatUint(uint64(activity.ID), 10),
			activity.SessionID,
			formatUintPointer(activity.MemberID),
			memberName,
			memberEmail,
			memberUsername,
			activity.APIAction,
			activity.APIPart,
			activity.Method,
			activity.APIStatus,
			formatStringPointer(activity.Description),
			formatStringPointer(activity.IPAddress),
			formatStringPointer(activity.UserAgent),
			formatStringPointer(activity.Referer),
			formatIntPointer(activity.StatusCode),
			formatStringPointer(activity.APIErrorMsg),
			formatStringPointer(activity.Metadata),
			formatStringPointer(activity.Role),
			formatStringPointer(activity.EventCategory),
			formatStringPointer(activity.EventName),
			formatStringPointer(activity.Country),
			formatStringPointer(activity.CountryCode),
			formatStringPointer(activity.Region),
			formatStringPointer(activity.City),
			formatStringPointer(activity.Timezone),
			formatFloatPointer(activity.Latitude),
			formatFloatPointer(activity.Longitude),
			externalPlatformIDs,
			strings.Join(externalPlatformNames, "|"),
			activity.CreatedAt.UTC().Format(time.RFC3339),
			activity.UpdatedAt.UTC().Format(time.RFC3339),
		}

		if err := writer.Write(row); err != nil {
			h.handleError(c, err)
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		h.handleError(c, err)
		return
	}

	filename := "user-activity-" + time.Now().UTC().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, "text/csv", csvBuffer.Bytes())
}

// GetEventCategories handles GET /user-activity/event-categories.
func (h *Handler) GetEventCategories(c *gin.Context) {
	if h.requireMember {
		if _, ok := h.getMemberID(c); !ok {
			h.handleError(c, useractivity.ErrUnauthorized)
			return
		}
	}

	categories, err := h.client.GetEventCategories(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, categories)
}

func (h *Handler) getMemberID(c *gin.Context) (uint, bool) {
	if h.memberIDFromContext == nil {
		return 0, false
	}
	return h.memberIDFromContext(c)
}

func (h *Handler) handleError(c *gin.Context, err error) {
	if h.errorHandler != nil {
		h.errorHandler(c, err)
		return
	}

	status := http.StatusInternalServerError
	switch {
	case err == useractivity.ErrBadRequest:
		status = http.StatusBadRequest
	case err == useractivity.ErrUnauthorized:
		status = http.StatusForbidden
	}

	c.JSON(status, gin.H{"error": err.Error()})
}

func formatStringPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatIntPointer(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func formatFloatPointer(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func formatUintPointer(value *uint) string {
	if value == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*value), 10)
}

func formatUintSlice(values []uint) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatUint(uint64(value), 10))
	}
	return strings.Join(parts, "|")
}
