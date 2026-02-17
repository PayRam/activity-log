package models

import "sync"

var (
	tablePrefix          string
	tablePrefixOnce      sync.Once
	activityLogTableName string
	activityLogTableOnce sync.Once
)

const (
	// DefaultActivityLogTableName is the default DB table used by the library.
	DefaultActivityLogTableName = "activity_logs"
)

// SetTablePrefix sets the global table prefix for all activity log models.
// This should be called once during application initialization.
func SetTablePrefix(prefix string) {
	tablePrefixOnce.Do(func() {
		tablePrefix = prefix
	})
}

// SetActivityLogTableName overrides the base activity log table name.
// This should be called once during application initialization.
func SetActivityLogTableName(tableName string) {
	activityLogTableOnce.Do(func() {
		activityLogTableName = tableName
	})
}

// GetTablePrefix returns the currently configured table prefix.
func GetTablePrefix() string {
	return tablePrefix
}

// GetActivityLogTableName returns the configured activity log base table name.
func GetActivityLogTableName() string {
	if activityLogTableName != "" {
		return activityLogTableName
	}
	return DefaultActivityLogTableName
}

// GetTableName returns the full table name with prefix.
func GetTableName(baseName string) string {
	if baseName == DefaultActivityLogTableName {
		baseName = GetActivityLogTableName()
	}
	return tablePrefix + baseName
}

// ResetTablePrefix resets table prefix and table name override (for testing only).
func ResetTablePrefix() {
	tablePrefix = ""
	tablePrefixOnce = sync.Once{}
	activityLogTableName = ""
	activityLogTableOnce = sync.Once{}
}
