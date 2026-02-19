package repositories

import (
	"reflect"

	"github.com/PayRam/activity-log/internal/models"
)

// CreateActivityLogParams contains fields required to persist a new activity log.
type CreateActivityLogParams struct {
	MemberID     *uint
	SessionID    string
	ProjectIDs   *[]uint
	Method       string
	APIPart      string
	APIStatus    string
	StatusCode   *int
	Description  *string
	IPAddress    *string
	UserAgent    *string
	Referer      *string
	APIAction    string
	APIErrorMsg  *string
	RequestBody  *string
	ResponseBody *string
	Metadata     *string

	Role          *string
	EventCategory *string
	EventName     *string

	Country     *string
	CountryCode *string
	Region      *string
	City        *string
	Timezone    *string
	Latitude    *float64
	Longitude   *float64
}

// UpdateActivityLogSessionParams contains updatable fields for a session-based update.
// ProjectIDs semantics:
// - nil pointer: do not update
// - pointer to nil slice: set DB value to NULL
// - pointer to empty/non-empty slice: set DB JSON array
// Other pointer fields follow standard semantics: nil means no update.
type UpdateActivityLogSessionParams struct {
	SessionID string

	ProjectIDs   *[]uint
	MemberID     *uint
	Method       *string
	APIPart      *string
	APIAction    *string
	APIStatus    *string
	StatusCode   *int
	Description  *string
	APIErrorMsg  *string
	IPAddress    *string
	UserAgent    *string
	Referer      *string
	ResponseBody *string
	Metadata     *string
	RequestBody  *string

	Role          *string
	EventCategory *string
	EventName     *string

	Country     *string
	CountryCode *string
	Region      *string
	City        *string
	Timezone    *string
	Latitude    *float64
	Longitude   *float64
}

// ActivityLogUpdateFields represents optional fields for session update operations.
// Nil fields are ignored by GORM.
type ActivityLogUpdateFields struct {
	ProjectIDs    *models.UintSlice `gorm:"column:project_ids;type:jsonb"`
	MemberID      *uint             `gorm:"column:member_id"`
	Method        *string           `gorm:"column:method"`
	APIPart       *string           `gorm:"column:api_part"`
	APIAction     *string           `gorm:"column:api_action"`
	APIStatus     *string           `gorm:"column:api_status"`
	StatusCode    *int              `gorm:"column:status_code"`
	Description   *string           `gorm:"column:description"`
	APIErrorMsg   *string           `gorm:"column:api_error_msg"`
	IPAddress     *string           `gorm:"column:ip_address"`
	UserAgent     *string           `gorm:"column:user_agent"`
	Referer       *string           `gorm:"column:referer"`
	ResponseBody  *string           `gorm:"column:response_body"`
	Metadata      *string           `gorm:"column:metadata"`
	RequestBody   *string           `gorm:"column:request_body"`
	Role          *string           `gorm:"column:role"`
	EventCategory *string           `gorm:"column:event_category"`
	EventName     *string           `gorm:"column:event_name"`
	Country       *string           `gorm:"column:country"`
	CountryCode   *string           `gorm:"column:country_code"`
	Region        *string           `gorm:"column:region"`
	City          *string           `gorm:"column:city"`
	Timezone      *string           `gorm:"column:timezone"`
	Latitude      *float64          `gorm:"column:latitude"`
	Longitude     *float64          `gorm:"column:longitude"`
}

// UpdateActivityLogSessionModel is the repository update payload.
type UpdateActivityLogSessionModel struct {
	SessionID string
	Fields    *ActivityLogUpdateFields
}

// HasValues returns true when at least one field is set.
func (f *ActivityLogUpdateFields) HasValues() bool {
	if f == nil {
		return false
	}

	v := reflect.ValueOf(*f)
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsNil() {
			return true
		}
	}

	return false
}
