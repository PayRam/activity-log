package useractivity

import internalutils "github.com/PayRam/activity-log/internal/utils"

// MergeMetadata merges patch fields into existing metadata JSON.
// If existing is non-JSON, it is preserved under "rawMetadata".
func MergeMetadata(existing *string, patch interface{}) *string {
	return internalutils.MergeMetadata(existing, patch)
}
