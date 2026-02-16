package models

import "sync"

var (
	tablePrefix     string
	tablePrefixOnce sync.Once
)

// SetTablePrefix sets the global table prefix for all user activity models.
// This should be called once during application initialization.
func SetTablePrefix(prefix string) {
	tablePrefixOnce.Do(func() {
		tablePrefix = prefix
	})
}

// GetTablePrefix returns the currently configured table prefix.
func GetTablePrefix() string {
	return tablePrefix
}

// GetTableName returns the full table name with prefix.
func GetTableName(baseName string) string {
	return tablePrefix + baseName
}

// ResetTablePrefix resets the table prefix (for testing only).
func ResetTablePrefix() {
	tablePrefix = ""
	tablePrefixOnce = sync.Once{}
}
