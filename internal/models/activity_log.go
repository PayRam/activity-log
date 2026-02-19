package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// UintSlice is a custom type for []uint that implements sql.Scanner and driver.Valuer.
type UintSlice []uint

// ActivityLog represents a single activity log record.
type ActivityLog struct {
	BaseModel

	// Actor
	MemberID *uint `gorm:"index"`

	// API request details
	SessionID    string    `gorm:"size:100;index"`
	ProjectIDs   UintSlice `gorm:"column:project_ids;type:jsonb"`
	Method       string    `gorm:"size:10;not null"`
	APIPart      string    `gorm:"size:255;not null"`
	APIStatus    string    `gorm:"size:50;index;not null"`
	StatusCode   *int      `gorm:"index"`
	Description  *string   `gorm:"size:255"`
	IPAddress    *string   `gorm:"size:50"`
	UserAgent    *string   `gorm:"size:500"`
	Referer      *string   `gorm:"size:500"`
	APIAction    string    `gorm:"size:100;index;not null"`
	APIErrorMsg  *string   `gorm:"size:1000"`
	RequestBody  *string   `gorm:"type:text"`
	ResponseBody *string   `gorm:"type:text"`
	Metadata     *string   `gorm:"type:json"`

	// Activity classification
	Role          *string `gorm:"size:100;index"`
	EventCategory *string `gorm:"size:100;index"`
	EventName     *string `gorm:"size:100;index"`

	// Location details (derived from IP address)
	Country     *string  `gorm:"size:100;index"`
	CountryCode *string  `gorm:"size:10"`
	Region      *string  `gorm:"size:100"`
	City        *string  `gorm:"size:100"`
	Timezone    *string  `gorm:"size:100"`
	Latitude    *float64 `gorm:"type:decimal(10,7)"`
	Longitude   *float64 `gorm:"type:decimal(10,7)"`

	// UpdateFields is prepared in service layer and consumed by repository update path.
	UpdateFields map[string]interface{} `gorm:"-" json:"-"`
}

// TableName applies the configured table prefix.
func (ActivityLog) TableName() string {
	return GetTableName(DefaultActivityLogTableName)
}

// Scan implements sql.Scanner for reading from database.
func (u *UintSlice) Scan(value interface{}) error {
	if value == nil {
		*u = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("UintSlice.Scan: unsupported value type %T", value)
	}

	if err := json.Unmarshal(bytes, u); err != nil {
		return fmt.Errorf("UintSlice.Scan: failed to unmarshal value: %w", err)
	}
	return nil
}

// Value implements driver.Valuer for writing to database.
func (u UintSlice) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	return json.Marshal(u)
}
