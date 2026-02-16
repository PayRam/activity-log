package models

import "sync"

var (
	tablePrefix           string
	tablePrefixOnce       sync.Once
	userActivityTableName string
	userActivityTableOnce sync.Once
)

const (
	// DefaultUserActivityTableName is the default DB table used by the library.
	DefaultUserActivityTableName = "user_activities"
)

// SetTablePrefix sets the global table prefix for all user activity models.
// This should be called once during application initialization.
func SetTablePrefix(prefix string) {
	tablePrefixOnce.Do(func() {
		tablePrefix = prefix
	})
}

// SetUserActivityTableName overrides the base user activity table name.
// This should be called once during application initialization.
func SetUserActivityTableName(tableName string) {
	userActivityTableOnce.Do(func() {
		userActivityTableName = tableName
	})
}

// GetTablePrefix returns the currently configured table prefix.
func GetTablePrefix() string {
	return tablePrefix
}

// GetUserActivityTableName returns the configured user activity base table name.
func GetUserActivityTableName() string {
	if userActivityTableName != "" {
		return userActivityTableName
	}
	return DefaultUserActivityTableName
}

// GetTableName returns the full table name with prefix.
func GetTableName(baseName string) string {
	if baseName == DefaultUserActivityTableName {
		baseName = GetUserActivityTableName()
	}
	return tablePrefix + baseName
}

// ResetTablePrefix resets table prefix and table name override (for testing only).
func ResetTablePrefix() {
	tablePrefix = ""
	tablePrefixOnce = sync.Once{}
	userActivityTableName = ""
	userActivityTableOnce = sync.Once{}
}
