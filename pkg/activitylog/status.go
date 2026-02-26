package activitylog

import "errors"

// APIStatus is the enum for API execution outcomes.
type APIStatus string

const (
	APIStatusSuccess APIStatus = "SUCCESS"
	APIStatusDenied  APIStatus = "DENIED"
	APIStatusError   APIStatus = "ERROR"
)

const (
	APIActionRead    = "READ"
	APIActionWrite   = "WRITE"
	APIActionDelete  = "DELETE"
	APIActionUnknown = "UNKNOWN"

	DefaultServiceMethod   = "SERVICE"
	DefaultServiceEndpoint = "service"
)

// String returns the string form of the status.
func (s APIStatus) String() string {
	return string(s)
}

// IsValid returns true when the status value is one of the supported enums.
func (s APIStatus) IsValid() bool {
	switch s {
	case APIStatusSuccess, APIStatusDenied, APIStatusError:
		return true
	default:
		return false
	}
}

// ErrorToAPIStatus maps a service-level error to an API status.
func ErrorToAPIStatus(err error) APIStatus {
	if err == nil {
		return APIStatusSuccess
	}
	if errors.Is(err, ErrUnauthorized) {
		return APIStatusDenied
	}
	return APIStatusError
}

// HTTPStatusCode is a typed HTTP status code used by the public API.
type HTTPStatusCode int

// IsValid returns true for valid HTTP status code ranges.
func (c HTTPStatusCode) IsValid() bool {
	return c >= 100 && c <= 599
}

// Int converts the typed status code to int.
func (c HTTPStatusCode) Int() int {
	return int(c)
}
