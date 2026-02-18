package useractivity

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ServiceOperation describes a service/repository operation to track.
type ServiceOperation struct {
	Name          string
	SessionID     string
	MemberID      *uint
	ProjectIDs    []uint
	Method        string
	Endpoint      string
	APIAction     string
	Description   *string
	Metadata      *string
	Role          *string
	EventCategory *string
	EventName     *string
}

// ServiceResult captures the outcome of a tracked operation.
type ServiceResult struct {
	Err error
}

// ServiceTrackerConfig configures a service-level tracker.
type ServiceTrackerConfig struct {
	Client  *Client
	Logger  *zap.Logger
	Async   bool
	OnError func(error)

	CreateEnricher func(context.Context, *CreateRequest)
	UpdateEnricher func(context.Context, *UpdateRequest, *ServiceResult)
}

// ServiceTracker logs activity for service/repository operations.
type ServiceTracker struct {
	client         *Client
	logger         *zap.Logger
	async          bool
	onError        func(error)
	createEnricher func(context.Context, *CreateRequest)
	updateEnricher func(context.Context, *UpdateRequest, *ServiceResult)
}

// NewServiceTracker creates a new service tracker instance.
func NewServiceTracker(cfg ServiceTrackerConfig) *ServiceTracker {
	if cfg.Client == nil {
		return &ServiceTracker{}
	}

	logger := cfg.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &ServiceTracker{
		client:         cfg.Client,
		logger:         logger,
		async:          cfg.Async,
		onError:        cfg.OnError,
		createEnricher: cfg.CreateEnricher,
		updateEnricher: cfg.UpdateEnricher,
	}
}

// Track executes the operation function and logs activity before/after it runs.
func (t *ServiceTracker) Track(ctx context.Context, op ServiceOperation, fn func(context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("operation function is nil")
	}
	if t == nil || t.client == nil {
		return fn(ctx)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sessionID := op.SessionID
	if sessionID == "" {
		if id, ok := SessionIDFromContext(ctx); ok {
			sessionID = id
		} else {
			sessionID = uuid.NewString()
		}
	}
	ctx = WithSessionID(ctx, sessionID)

	method := op.Method
	if method == "" {
		method = DefaultServiceMethod
	}

	endpoint := op.Endpoint
	if endpoint == "" {
		endpoint = op.Name
	}
	if endpoint == "" {
		endpoint = DefaultServiceEndpoint
	}

	action := op.APIAction
	if action == "" {
		action = APIActionUnknown
	}

	baseTrail := operationTrailFromContext(ctx)
	currentStep := OperationTrailEntry{
		Name:      endpoint,
		APIAction: action,
		Method:    method,
		Endpoint:  endpoint,
		Status:    "STARTED",
	}
	createTrail := append(slices.Clone(baseTrail), currentStep)

	metadata := op.Metadata
	metadata = mergeServiceMetadata(metadata, ServiceMetadata{
		Source:         "SERVICE",
		OperationName:  endpoint,
		OperationTrail: createTrail,
	})

	createReq := CreateRequest{
		SessionID:     sessionID,
		MemberID:      op.MemberID,
		ProjectIDs:    op.ProjectIDs,
		Method:        method,
		Endpoint:      endpoint,
		APIAction:     action,
		APIStatus:     APIStatusSuccess,
		Description:   op.Description,
		Metadata:      metadata,
		Role:          op.Role,
		EventCategory: op.EventCategory,
		EventName:     op.EventName,
	}

	if t.createEnricher != nil {
		t.createEnricher(ctx, &createReq)
	}

	created := true
	if t.async {
		go func() {
			if _, err := t.client.CreateActivityLogs(context.Background(), createReq); err != nil {
				t.handleError(err)
			}
		}()
	} else {
		if _, err := t.client.CreateActivityLogs(ctx, createReq); err != nil {
			t.handleError(err)
			created = false
		}
	}

	callCtx := withOperationTrail(ctx, createTrail)
	opErr := fn(callCtx)
	if !created && !t.async {
		return opErr
	}

	status := ErrorToAPIStatus(opErr)
	finalStep := currentStep
	finalStep.Status = string(status)
	finalTrail := append(slices.Clone(baseTrail), finalStep)

	updateReq := UpdateRequest{
		SessionID: sessionID,
		APIStatus: &status,
		Metadata: mergeServiceMetadata(op.Metadata, ServiceMetadata{
			Source:         "SERVICE",
			OperationName:  endpoint,
			OperationTrail: finalTrail,
		}),
	}

	result := &ServiceResult{Err: opErr}
	if t.updateEnricher != nil {
		t.updateEnricher(ctx, &updateReq, result)
	}

	if t.async {
		go func(req UpdateRequest) {
			if _, err := t.client.UpdateActivityLogSessionID(context.Background(), req); err != nil {
				t.handleError(err)
			}
		}(updateReq)
		return opErr
	}

	if _, err := t.client.UpdateActivityLogSessionID(ctx, updateReq); err != nil {
		t.handleError(err)
	}

	return opErr
}

func (t *ServiceTracker) handleError(err error) {
	if err == nil {
		return
	}
	if t.onError != nil {
		t.onError(err)
		return
	}
	if t.logger != nil {
		t.logger.Error("useractivity service tracker error", zap.Error(err))
	}
}
