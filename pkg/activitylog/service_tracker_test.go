package activitylog

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
)

type trackerStubService struct {
	createErr error
	updateErr error

	mu      sync.Mutex
	created []*models.ActivityLog
	updated []*models.ActivityLog
}

func (s *trackerStubService) CreateActivityLogs(ctx context.Context, params repositories.CreateActivityLogParams) (*models.ActivityLog, error) {
	activity := trackerCreateModelFromParams(params)
	s.mu.Lock()
	s.created = append(s.created, activity)
	s.mu.Unlock()
	if s.createErr != nil {
		return nil, s.createErr
	}
	return activity, nil
}

func (s *trackerStubService) UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	activity := trackerUpdateModelFromParams(params)
	s.mu.Lock()
	s.updated = append(s.updated, activity)
	s.mu.Unlock()
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return activity, nil
}

func (s *trackerStubService) GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (s *trackerStubService) GetEventCategories(ctx context.Context) ([]string, error) {
	return nil, nil
}

func trackerCreateModelFromParams(params repositories.CreateActivityLogParams) *models.ActivityLog {
	activity := &models.ActivityLog{
		MemberID:      params.MemberID,
		SessionID:     params.SessionID,
		Method:        params.Method,
		APIPart:       params.APIPart,
		APIStatus:     params.APIStatus,
		StatusCode:    params.StatusCode,
		Description:   params.Description,
		IPAddress:     params.IPAddress,
		UserAgent:     params.UserAgent,
		Referer:       params.Referer,
		APIAction:     params.APIAction,
		APIErrorMsg:   params.APIErrorMsg,
		RequestBody:   params.RequestBody,
		ResponseBody:  params.ResponseBody,
		Metadata:      params.Metadata,
		Role:          params.Role,
		EventCategory: params.EventCategory,
		EventName:     params.EventName,
		Country:       params.Country,
		CountryCode:   params.CountryCode,
		Region:        params.Region,
		City:          params.City,
		Timezone:      params.Timezone,
		Latitude:      params.Latitude,
		Longitude:     params.Longitude,
	}
	if params.ProjectIDs != nil {
		activity.ProjectIDs = models.UintSlice(params.ProjectIDs)
	}
	return activity
}

func trackerUpdateModelFromParams(params repositories.UpdateActivityLogSessionParams) *models.ActivityLog {
	activity := &models.ActivityLog{SessionID: params.SessionID}
	if params.ProjectIDs != nil {
		activity.ProjectIDs = models.UintSlice(params.ProjectIDs)
	}
	if params.MemberID != nil {
		activity.MemberID = params.MemberID
	}
	if params.Method != nil {
		activity.Method = *params.Method
	}
	if params.APIPart != nil {
		activity.APIPart = *params.APIPart
	}
	if params.APIAction != nil {
		activity.APIAction = *params.APIAction
	}
	if params.APIStatus != nil {
		activity.APIStatus = *params.APIStatus
	}
	activity.StatusCode = params.StatusCode
	activity.Description = params.Description
	activity.APIErrorMsg = params.APIErrorMsg
	activity.IPAddress = params.IPAddress
	activity.UserAgent = params.UserAgent
	activity.Referer = params.Referer
	activity.ResponseBody = params.ResponseBody
	activity.Metadata = params.Metadata
	activity.RequestBody = params.RequestBody
	activity.Role = params.Role
	activity.EventCategory = params.EventCategory
	activity.EventName = params.EventName
	activity.Country = params.Country
	activity.CountryCode = params.CountryCode
	activity.Region = params.Region
	activity.City = params.City
	activity.Timezone = params.Timezone
	activity.Latitude = params.Latitude
	activity.Longitude = params.Longitude
	return activity
}

func TestServiceTrackerTrackSuccess(t *testing.T) {
	stub := &trackerStubService{}
	client := &Client{svc: stub}
	tracker := NewServiceTracker(ServiceTrackerConfig{Client: client})

	memberID := uint(42)
	op := ServiceOperation{
		Name:      "PaymentRequestService.Create",
		MemberID:  &memberID,
		APIAction: APIActionWrite,
	}

	if err := tracker.Track(context.Background(), op, func(ctx context.Context) error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.created) != 1 || len(stub.updated) != 1 {
		t.Fatalf("expected create/update calls, got %d/%d", len(stub.created), len(stub.updated))
	}

	created := stub.created[0]
	if created.Method != DefaultServiceMethod {
		t.Fatalf("expected default method %q, got %q", DefaultServiceMethod, created.Method)
	}
	if created.APIPart != op.Name {
		t.Fatalf("expected endpoint %q, got %q", op.Name, created.APIPart)
	}
	if created.APIAction != string(APIActionWrite) {
		t.Fatalf("expected API action %q, got %q", APIActionWrite, created.APIAction)
	}

	var createMetadata ServiceMetadata
	if created.Metadata == nil || json.Unmarshal([]byte(*created.Metadata), &createMetadata) != nil {
		t.Fatalf("expected valid create metadata")
	}
	if len(createMetadata.OperationTrail) != 1 || createMetadata.OperationTrail[0].Status != "STARTED" {
		t.Fatalf("expected single STARTED trail entry")
	}

	updated := stub.updated[0]
	if updated.APIStatus != string(APIStatusSuccess) {
		t.Fatalf("expected update status %q, got %q", APIStatusSuccess, updated.APIStatus)
	}
	if updated.SessionID != created.SessionID {
		t.Fatalf("expected matching session IDs")
	}

	var updateMetadata ServiceMetadata
	if updated.Metadata == nil || json.Unmarshal([]byte(*updated.Metadata), &updateMetadata) != nil {
		t.Fatalf("expected valid update metadata")
	}
	if len(updateMetadata.OperationTrail) != 1 || updateMetadata.OperationTrail[0].Status != string(APIStatusSuccess) {
		t.Fatalf("expected final trail status %q", APIStatusSuccess)
	}
}

func TestServiceTrackerTrackErrorAndUnauthorized(t *testing.T) {
	stub := &trackerStubService{}
	client := &Client{svc: stub}
	tracker := NewServiceTracker(ServiceTrackerConfig{Client: client})

	err := tracker.Track(context.Background(), ServiceOperation{
		Endpoint:  "DepositService.Create",
		APIAction: APIActionWrite,
	}, func(ctx context.Context) error {
		return errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected operation error")
	}
	if len(stub.updated) == 0 || stub.updated[0].APIStatus != string(APIStatusError) {
		t.Fatalf("expected update status %q", APIStatusError)
	}

	stub = &trackerStubService{}
	client = &Client{svc: stub}
	tracker = NewServiceTracker(ServiceTrackerConfig{Client: client})
	err = tracker.Track(context.Background(), ServiceOperation{
		Endpoint:  "DepositService.Get",
		APIAction: APIActionRead,
	}, func(ctx context.Context) error {
		return ErrUnauthorized
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized")
	}
	if len(stub.updated) == 0 || stub.updated[0].APIStatus != string(APIStatusDenied) {
		t.Fatalf("expected update status %q", APIStatusDenied)
	}
}

func TestServiceTrackerUsesContextSessionAndNestedTrail(t *testing.T) {
	stub := &trackerStubService{}
	client := &Client{svc: stub}
	tracker := NewServiceTracker(ServiceTrackerConfig{Client: client})

	rootSession := "api-session-1"
	ctx := WithSessionID(context.Background(), rootSession)

	err := tracker.Track(ctx, ServiceOperation{
		Name:      "PaymentRequestService.Create",
		APIAction: APIActionWrite,
	}, func(ctx context.Context) error {
		return tracker.Track(ctx, ServiceOperation{
			Name:      "DepositService.Create",
			APIAction: APIActionWrite,
		}, func(context.Context) error { return nil })
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.created) != 2 || len(stub.updated) != 2 {
		t.Fatalf("expected two create and two update calls, got %d/%d", len(stub.created), len(stub.updated))
	}

	for _, created := range stub.created {
		if created.SessionID != rootSession {
			t.Fatalf("expected session propagation, got %q", created.SessionID)
		}
	}
	for _, updated := range stub.updated {
		if updated.SessionID != rootSession {
			t.Fatalf("expected session propagation on update, got %q", updated.SessionID)
		}
	}

	var nestedCreateMetadata ServiceMetadata
	if stub.created[1].Metadata == nil || json.Unmarshal([]byte(*stub.created[1].Metadata), &nestedCreateMetadata) != nil {
		t.Fatalf("expected valid nested create metadata")
	}
	if len(nestedCreateMetadata.OperationTrail) != 2 {
		t.Fatalf("expected nested trail depth 2, got %d", len(nestedCreateMetadata.OperationTrail))
	}
	if nestedCreateMetadata.OperationTrail[0].Name != "PaymentRequestService.Create" {
		t.Fatalf("expected parent operation in trail")
	}
	if nestedCreateMetadata.OperationTrail[1].Name != "DepositService.Create" {
		t.Fatalf("expected nested operation in trail")
	}
}

func TestServiceTrackerCreateFailsSkipsUpdate(t *testing.T) {
	stub := &trackerStubService{createErr: errors.New("create failed")}
	client := &Client{svc: stub}
	tracker := NewServiceTracker(ServiceTrackerConfig{Client: client})

	if err := tracker.Track(context.Background(), ServiceOperation{
		Endpoint:  "MemberRepo.Delete",
		APIAction: APIActionDelete,
	}, func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.updated) != 0 {
		t.Fatalf("expected no update when create fails")
	}
}

func TestServiceTrackerMergesCustomMetadata(t *testing.T) {
	stub := &trackerStubService{}
	client := &Client{svc: stub}
	tracker := NewServiceTracker(ServiceTrackerConfig{Client: client})

	custom := `{"workflow":"payment_request","attempt":1}`
	err := tracker.Track(context.Background(), ServiceOperation{
		Name:      "DepositService.Update",
		APIAction: APIActionWrite,
		Metadata:  &custom,
	}, func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.created) != 1 || stub.created[0].Metadata == nil {
		t.Fatalf("expected create metadata")
	}
	if len(stub.updated) != 1 || stub.updated[0].Metadata == nil {
		t.Fatalf("expected update metadata")
	}

	var createPayload map[string]interface{}
	if err := json.Unmarshal([]byte(*stub.created[0].Metadata), &createPayload); err != nil {
		t.Fatalf("expected json metadata: %v", err)
	}
	if createPayload["workflow"] != "payment_request" {
		t.Fatalf("expected custom metadata key in create payload")
	}
	if _, ok := createPayload["operationTrail"]; !ok {
		t.Fatalf("expected operationTrail in create payload")
	}

	var updatePayload map[string]interface{}
	if err := json.Unmarshal([]byte(*stub.updated[0].Metadata), &updatePayload); err != nil {
		t.Fatalf("expected json metadata: %v", err)
	}
	if updatePayload["workflow"] != "payment_request" {
		t.Fatalf("expected custom metadata key in update payload")
	}
	if _, ok := updatePayload["operationTrail"]; !ok {
		t.Fatalf("expected operationTrail in update payload")
	}
}

func TestServiceTrackerNilClientAndNilFn(t *testing.T) {
	tracker := NewServiceTracker(ServiceTrackerConfig{})
	called := false

	if err := tracker.Track(context.Background(), ServiceOperation{}, func(ctx context.Context) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected operation execution")
	}

	if err := tracker.Track(context.Background(), ServiceOperation{}, nil); err == nil {
		t.Fatalf("expected error for nil fn")
	}
}
