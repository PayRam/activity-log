package repositories

import (
	"fmt"
	"time"

	"github.com/PayRam/activity-log/internal/models"
	"gorm.io/gorm"
)

// UserActivityFilters defines filter options for listing activities.
type UserActivityFilters struct {
	SessionID           *string
	Search              *string
	StatusCodes         []int
	IDS                 []uint
	MemberIDs           []uint
	Methods             []string
	APIStatuses         []string
	EventCategories     []string
	EventNames          []string
	IPAddresses         []string
	Countries           []string
	Roles               []string
	ExternalPlatformIDs []uint
	ProjectFilter       *string

	Limit         *int
	Offset        *int
	SortBy        *string
	Order         *string
	GreaterThanID *uint
	LessThanID    *uint
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time
	StartDate     *time.Time
	EndDate       *time.Time
}

var allowedUserActivitySortColumns = map[string]bool{
	"id":             true,
	"created_at":     true,
	"updated_at":     true,
	"member_id":      true,
	"session_id":     true,
	"method":         true,
	"api_status":     true,
	"status_code":    true,
	"event_name":     true,
	"event_category": true,
	"ip_address":     true,
	"country":        true,
	"city":           true,
}

func ApplyUserActivityGetFilters(query *gorm.DB, filter UserActivityFilters) *gorm.DB {
	tableName := models.GetTableName(models.DefaultUserActivityTableName)

	if len(filter.IDS) > 0 {
		query = query.Where(fmt.Sprintf("%s.id IN ?", tableName), filter.IDS)
	}
	if len(filter.Methods) > 0 {
		query = query.Where(fmt.Sprintf("%s.method IN ?", tableName), filter.Methods)
	}
	if len(filter.MemberIDs) > 0 {
		query = query.Where(fmt.Sprintf("%s.member_id IN ?", tableName), filter.MemberIDs)
	}
	if len(filter.APIStatuses) > 0 {
		query = query.Where(fmt.Sprintf("%s.api_status IN ?", tableName), filter.APIStatuses)
	}
	if len(filter.EventNames) > 0 {
		query = query.Where(fmt.Sprintf("%s.event_name IN ?", tableName), filter.EventNames)
	}
	if len(filter.EventCategories) > 0 {
		query = query.Where(fmt.Sprintf("%s.event_category IN ?", tableName), filter.EventCategories)
	}

	if len(filter.StatusCodes) > 0 {
		query = query.Where(fmt.Sprintf("%s.status_code IN ?", tableName), filter.StatusCodes)
	}
	if filter.Search != nil {
		searchPattern := "%" + *filter.Search + "%"
		query = query.Where(fmt.Sprintf(`(
			%s.member_id IN (
				SELECT id FROM members 
				WHERE email LIKE ? OR name LIKE ? OR username LIKE ?
			) OR
			%s.method LIKE ? OR
			%s.country LIKE ? OR
			%s.region LIKE ? OR
			%s.city LIKE ? OR
			%s.description LIKE ? OR
			%s.api_part LIKE ? OR
			%s.event_name LIKE ? OR
			%s.event_category LIKE ? OR
			%s.api_error_msg LIKE ? OR
			%s.session_id LIKE ? OR
			%s.ip_address LIKE ? OR
			%s.user_agent LIKE ?
		)`, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName),
			searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}
	if filter.SessionID != nil {
		query = query.Where(fmt.Sprintf("%s.session_id = ?", tableName), *filter.SessionID)
	}

	if len(filter.IPAddresses) > 0 {
		query = query.Where(fmt.Sprintf("%s.ip_address IN ?", tableName), filter.IPAddresses)
	}
	if len(filter.Countries) > 0 {
		query = query.Where(fmt.Sprintf("%s.country IN ?", tableName), filter.Countries)
	}
	if len(filter.Roles) > 0 {
		query = query.Where(fmt.Sprintf("%s.role IN ?", tableName), filter.Roles)
	}

	if filter.ProjectFilter != nil {
		switch *filter.ProjectFilter {
		case "NO_IDS":
			query = query.Where("(external_platform_ids IS NULL OR external_platform_ids::jsonb = '[]'::jsonb OR external_platform_ids::jsonb = 'null'::jsonb)")
		case "ALL":
			query = query.Where("(external_platform_ids IS NOT NULL AND external_platform_ids::jsonb != '[]'::jsonb AND external_platform_ids::jsonb != 'null'::jsonb)")
		}
	} else if len(filter.ExternalPlatformIDs) > 0 {
		for _, id := range filter.ExternalPlatformIDs {
			query = query.Where("external_platform_ids::jsonb @> ?::jsonb", fmt.Sprintf("[%d]", id))
		}
	}

	if filter.StartDate != nil {
		query = query.Where(fmt.Sprintf("%s.created_at >= ?", tableName), *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where(fmt.Sprintf("%s.created_at <= ?", tableName), *filter.EndDate)
	}

	return query
}

func ApplyUserActivitiesPaginationConditions(query *gorm.DB, filter UserActivityFilters) *gorm.DB {
	if filter.Limit != nil && *filter.Limit > 0 {
		query = query.Limit(*filter.Limit)
	}
	if filter.Offset != nil && *filter.Offset > 0 {
		query = query.Offset(*filter.Offset)
	}
	if filter.Order != nil {
		order := "DESC"
		if *filter.Order == "ASC" {
			order = "ASC"
		}
		sortColumn := "id"
		if filter.SortBy != nil {
			if allowedUserActivitySortColumns[*filter.SortBy] {
				sortColumn = *filter.SortBy
			}
		}
		query = query.Order(fmt.Sprintf("%s %s", sortColumn, order))
	}
	if filter.GreaterThanID != nil {
		query = query.Where("id > ?", *filter.GreaterThanID)
	}
	if filter.LessThanID != nil {
		query = query.Where("id < ?", *filter.LessThanID)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if filter.CreatedBefore != nil {
		query = query.Where("created_at <= ?", *filter.CreatedBefore)
	}
	if filter.UpdatedAfter != nil {
		query = query.Where("updated_at >= ?", *filter.UpdatedAfter)
	}
	if filter.UpdatedBefore != nil {
		query = query.Where("updated_at <= ?", *filter.UpdatedBefore)
	}

	return query
}
