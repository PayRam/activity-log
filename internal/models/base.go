package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel provides standard primary key and timestamp fields.
type BaseModel struct {
	ID        uint           `gorm:"primaryKey"`
	CreatedAt time.Time      `gorm:"index"`
	UpdatedAt time.Time      `gorm:"index"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
