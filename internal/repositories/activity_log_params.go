package repositories

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

// UpdateActivityLogSessionModel is the repository update payload.
type UpdateActivityLogSessionModel struct {
	SessionID string
	Updates   map[string]interface{}
}
