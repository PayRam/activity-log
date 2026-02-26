package repositories

import (
	"fmt"
	"time"

	"github.com/PayRam/activity-log/internal/models"
	"gorm.io/gorm"
)

type ActionStatusPairFilter struct {
	APIAction string
	APIStatus string
}

// ActivityLogFilters defines filter options for listing activities.
type ActivityLogFilters struct {
	SessionIDs               []string
	Search                   *string
	StatusCodes              []int
	IDS                      []uint
	MemberIDs                []uint
	Methods                  []string
	ExcludeMethods           []string
	APIStatuses              []string
	ExcludeAPIStatuses       []string
	ExcludeActionStatusPairs []ActionStatusPairFilter
	EventCategories          []string
	EventNames               []string
	IPAddresses              []string
	Countries                []string
	Roles                    []string
	// ProjectIDs filter semantics:
	// nil => ignore project filtering.
	// [] => logs with no project IDs.
	// [id1, id2, ...] => logs containing provided IDs, plus rows with no project IDs.
	ProjectIDs []uint

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

var allowedActivityLogSortColumns = map[string]bool{
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

func ApplyActivityLogGetFilters(query *gorm.DB, filter ActivityLogFilters) *gorm.DB {
	tableName := models.GetTableName(models.DefaultActivityLogTableName)

	if len(filter.IDS) > 0 {
		query = query.Where(fmt.Sprintf("%s.id IN ?", tableName), filter.IDS)
	}
	if len(filter.Methods) > 0 {
		query = query.Where(fmt.Sprintf("%s.method IN ?", tableName), filter.Methods)
	}
	if len(filter.ExcludeMethods) > 0 {
		query = query.Where(fmt.Sprintf("%s.method NOT IN ?", tableName), filter.ExcludeMethods)
	}
	if len(filter.MemberIDs) > 0 {
		query = query.Where(fmt.Sprintf("%s.member_id IN ?", tableName), filter.MemberIDs)
	}
	if len(filter.APIStatuses) > 0 {
		query = query.Where(fmt.Sprintf("%s.api_status IN ?", tableName), filter.APIStatuses)
	}
	if len(filter.ExcludeAPIStatuses) > 0 {
		query = query.Where(fmt.Sprintf("%s.api_status NOT IN ?", tableName), filter.ExcludeAPIStatuses)
	}
	for _, pair := range filter.ExcludeActionStatusPairs {
		if pair.APIAction == "" || pair.APIStatus == "" {
			continue
		}
		query = query.Where(
			fmt.Sprintf("NOT (%s.api_action = ? AND %s.api_status = ?)", tableName, tableName),
			pair.APIAction,
			pair.APIStatus,
		)
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
	if len(filter.SessionIDs) > 0 {
		query = query.Where(fmt.Sprintf("%s.session_id IN ?", tableName), filter.SessionIDs)
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

	if filter.ProjectIDs != nil {
		if len(filter.ProjectIDs) == 0 {
			query = query.Where("(project_ids IS NULL OR project_ids::jsonb = '[]'::jsonb OR project_ids::jsonb = 'null'::jsonb)")
		} else {
			for _, id := range filter.ProjectIDs {
				query = query.Where(
					"(project_ids::jsonb @> ?::jsonb OR project_ids IS NULL OR project_ids::jsonb = '[]'::jsonb OR project_ids::jsonb = 'null'::jsonb)",
					fmt.Sprintf("[%d]", id),
				)
			}
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

func ApplyActivityLogsPaginationConditions(query *gorm.DB, filter ActivityLogFilters) *gorm.DB {
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
			if allowedActivityLogSortColumns[*filter.SortBy] {
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
