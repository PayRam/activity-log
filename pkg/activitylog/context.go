package activitylog

import "context"

type contextKey string

const (
	contextKeySessionID      contextKey = "activitylog_session_id"
	contextKeyOperationTrail contextKey = "activitylog_operation_trail"
)

// WithSessionID stores the activity session ID on the context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if sessionID == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKeySessionID, sessionID)
}

// SessionIDFromContext reads the activity session ID from context.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	sessionID, ok := ctx.Value(contextKeySessionID).(string)
	if !ok || sessionID == "" {
		return "", false
	}
	return sessionID, true
}

// OperationTrailEntry is a single service-level operation in the metadata trail.
type OperationTrailEntry struct {
	Name      string `json:"name"`
	APIAction string `json:"apiAction,omitempty"`
	Method    string `json:"method,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	Status    string `json:"status,omitempty"`
}

func withOperationTrail(ctx context.Context, trail []OperationTrailEntry) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(trail) == 0 {
		return ctx
	}
	copyTrail := append([]OperationTrailEntry(nil), trail...)
	return context.WithValue(ctx, contextKeyOperationTrail, copyTrail)
}

func operationTrailFromContext(ctx context.Context) []OperationTrailEntry {
	if ctx == nil {
		return nil
	}
	trail, ok := ctx.Value(contextKeyOperationTrail).([]OperationTrailEntry)
	if !ok || len(trail) == 0 {
		return nil
	}
	return append([]OperationTrailEntry(nil), trail...)
}
