package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultMaxBodyBytes int64 = 1 * 1024 * 1024

// NormalizeMaxBodyBytes returns a safe max body size.
func NormalizeMaxBodyBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return defaultMaxBodyBytes
	}
	return maxBytes
}

// MethodToAction maps HTTP method to an API action.
func MethodToAction(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "READ"
	case "POST", "PUT", "PATCH":
		return "WRITE"
	case "DELETE":
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// StatusToAPIStatus maps HTTP status to API status.
func StatusToAPIStatus(status int) string {
	switch {
	case status >= 500:
		return "ERROR"
	case status >= 400:
		return "DENIED"
	default:
		return "SUCCESS"
	}
}

// ReadRequestBody reads and restores the request body.
func ReadRequestBody(r *http.Request, maxBytes int64, redact func([]byte) []byte) (*string, error) {
	if r == nil || r.Body == nil {
		return nil, nil
	}

	limited := io.LimitReader(r.Body, maxBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	// Restore original body for downstream handlers.
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if len(bodyBytes) == 0 {
		return nil, nil
	}

	if redact != nil {
		bodyBytes = redact(bodyBytes)
	}

	bodyStr := string(bodyBytes)
	return &bodyStr, nil
}

// BodyCapture captures response body up to a max size.
type BodyCapture struct {
	max       int64
	truncated bool
	buf       bytes.Buffer
}

func NewBodyCapture(maxBytes int64) *BodyCapture {
	return &BodyCapture{max: maxBytes}
}

func (b *BodyCapture) Write(p []byte) {
	if b == nil || b.max <= 0 {
		return
	}
	remaining := b.max - int64(b.buf.Len())
	if remaining <= 0 {
		b.truncated = true
		return
	}

	if int64(len(p)) <= remaining {
		_, _ = b.buf.Write(p)
		return
	}

	_, _ = b.buf.Write(p[:remaining])
	b.truncated = true
}

func (b *BodyCapture) String() string {
	if b == nil {
		return ""
	}
	if !b.truncated {
		return b.buf.String()
	}
	return fmt.Sprintf("%s...", b.buf.String())
}
