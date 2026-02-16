package useractivity

import "time"

// Activity represents a persisted user activity record.
type Activity struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time

	MemberID *uint

	SessionID           string
	ExternalPlatformIDs []uint
	Method              string
	APIPart             string
	APIStatus           string
	StatusCode          *int
	Description         *string
	IPAddress           *string
	UserAgent           *string
	Referer             *string
	APIAction           string
	APIErrorMsg         *string
	RequestBody         *string
	ResponseBody        *string
	Metadata            *string

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

	Member            *MemberInfo
	ExternalPlatforms []ExternalPlatformInfo
}

// CreateRequest defines the fields for creating a new activity record.
type CreateRequest struct {
	MemberID            *uint
	SessionID           string
	ExternalPlatformIDs []uint
	Method              string
	Endpoint            string
	APIAction           string
	APIStatus           string
	StatusCode          *int
	Description         *string
	APIErrorMsg         *string
	IPAddress           *string
	UserAgent           *string
	Referer             *string
	RequestBody         *string
	ResponseBody        *string
	Metadata            *string

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

// UpdateRequest defines the fields for updating an activity record by session ID.
type UpdateRequest struct {
	SessionID           string
	ExternalPlatformIDs []uint
	MemberID            *uint
	Method              *string
	Endpoint            *string
	APIAction           *string
	APIStatus           *string
	StatusCode          *int
	Description         *string
	APIErrorMsg         *string
	IPAddress           *string
	UserAgent           *string
	Referer             *string
	ResponseBody        *string
	Metadata            *string
	RequestBody         *string

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

// MemberInfo represents a lightweight member record for responses.
type MemberInfo struct {
	ID       uint
	Name     string
	Email    *string
	Username *string
}

// ExternalPlatformInfo represents a lightweight external platform record for responses.
type ExternalPlatformInfo struct {
	ID       uint
	Name     string
	LogoPath string
}

// PaginationConditions defines filtering/pagination controls.
type PaginationConditions struct {
	Limit         *int       `form:"limit"`
	Offset        *int       `form:"offset"`
	SortBy        *string    `form:"sortBy"`
	Order         *string    `form:"order"`
	GreaterThanID *uint      `form:"greaterThanID"`
	LessThanID    *uint      `form:"lessThanID"`
	CreatedAfter  *time.Time `form:"createdAfter"`
	CreatedBefore *time.Time `form:"createdBefore"`
	UpdatedAfter  *time.Time `form:"updatedAfter"`
	UpdatedBefore *time.Time `form:"updatedBefore"`
	StartDate     *time.Time `form:"startDate"`
	EndDate       *time.Time `form:"endDate"`
}

// GetRequest defines filters for getting user activities.
type GetRequest struct {
	StatusCode          *int     `form:"statusCode"`
	Search              *string  `form:"search"`
	SessionID           *string  `form:"sessionID"`
	EventCategories     []string `form:"eventCategories"`
	Methods             []string `form:"methods"`
	EventNames          []string `form:"eventNames"`
	IDS                 []uint   `form:"ids"`
	MemberIDs           []uint   `form:"memberIDs"`
	ExternalPlatformIDs []uint   `form:"externalPlatformIDs"`
	ProjectFilter       *string  `form:"projectFilter"`
	APIStatuses         []string `form:"apiStatuses"`
	IPAddresses         []string `form:"ipAddresses"`
	Countries           []string `form:"countries"`
	Roles               []string `form:"roles"`

	Export bool `form:"-"`

	PaginationConditions
}

// GetResponse contains activities and total count.
type GetResponse struct {
	Activities []Activity
	TotalCount int64
}
