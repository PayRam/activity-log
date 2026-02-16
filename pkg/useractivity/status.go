package useractivity

import "errors"

const (
	APIStatusSuccess = "SUCCESS"
	APIStatusDenied  = "DENIED"
	APIStatusError   = "ERROR"

	APIActionRead    = "READ"
	APIActionWrite   = "WRITE"
	APIActionDelete  = "DELETE"
	APIActionUnknown = "UNKNOWN"

	DefaultServiceMethod   = "SERVICE"
	DefaultServiceEndpoint = "service"
)

// ErrorToAPIStatus maps a service-level error to an API status.
func ErrorToAPIStatus(err error) string {
	if err == nil {
		return APIStatusSuccess
	}
	if errors.Is(err, ErrUnauthorized) {
		return APIStatusDenied
	}
	return APIStatusError
}
