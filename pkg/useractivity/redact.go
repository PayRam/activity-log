package useractivity

import internalutils "github.com/PayRam/user-activity-go/internal/utils"

// RedactDefaultJSONKeys redacts common sensitive keys in a JSON payload.
// If payload is not valid JSON, it is returned unchanged.
func RedactDefaultJSONKeys(payload []byte) []byte {
	return internalutils.RedactJSONKeys(payload, internalutils.DefaultRedactKeys()...)
}

// RedactJSONKeys redacts the provided keys in a JSON payload.
// If payload is not valid JSON, it is returned unchanged.
func RedactJSONKeys(payload []byte, keys ...string) []byte {
	return internalutils.RedactJSONKeys(payload, keys...)
}
