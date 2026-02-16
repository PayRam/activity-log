package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// UintSlice is a custom type for []uint that implements sql.Scanner and driver.Valuer.
type UintSlice []uint

// UserActivity represents a single user activity record.
type UserActivity struct {
	BaseModel

	// Actor
	MemberID *uint `gorm:"index"`

	// API request details
	SessionID           string    `gorm:"size:100;index"`
	ExternalPlatformIDs UintSlice `gorm:"type:jsonb"`
	Method              string    `gorm:"size:10;not null"`
	APIPart             string    `gorm:"size:255;not null"`
	APIStatus           string    `gorm:"size:50;index;not null"`
	StatusCode          *int      `gorm:"index"`
	Description         *string   `gorm:"size:255"`
	IPAddress           *string   `gorm:"size:50"`
	UserAgent           *string   `gorm:"size:500"`
	Referer             *string   `gorm:"size:500"`
	APIAction           string    `gorm:"size:100;index;not null"`
	APIErrorMsg         *string   `gorm:"size:1000"`
	RequestBody         *string   `gorm:"type:text"`
	ResponseBody        *string   `gorm:"type:text"`
	Metadata            *string   `gorm:"type:json"`

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
}

// TableName applies the configured table prefix.
func (UserActivity) TableName() string {
	return GetTableName(DefaultUserActivityTableName)
}

// Scan implements sql.Scanner for reading from database.
func (u *UintSlice) Scan(value interface{}) error {
	if value == nil {
		*u = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal UintSlice value: %v", value)
	}
	return json.Unmarshal(bytes, u)
}

// Value implements driver.Valuer for writing to database.
func (u UintSlice) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	return json.Marshal(u)
}
