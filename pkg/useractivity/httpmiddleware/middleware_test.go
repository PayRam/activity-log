package httpmiddleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/PayRam/activity-log/internal/models"
	"github.com/PayRam/activity-log/internal/repositories"
	"github.com/PayRam/activity-log/internal/services"
	"github.com/PayRam/activity-log/pkg/useractivity"
)

type stubService struct {
	createErr error
	updateErr error

	mu      sync.Mutex
	created []*models.ActivityLog
	updated []*models.ActivityLog

	createCh chan struct{}
	updateCh chan struct{}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonRoundTripResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func createActivityModelFromParams(params repositories.CreateActivityLogParams) *models.ActivityLog {
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
		activity.ProjectIDs = models.UintSlice(*params.ProjectIDs)
	}
	return activity
}

func updateActivityModelFromParams(params repositories.UpdateActivityLogSessionParams) *models.ActivityLog {
	activity := &models.ActivityLog{SessionID: params.SessionID}
	if params.ProjectIDs != nil {
		activity.ProjectIDs = models.UintSlice(*params.ProjectIDs)
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

func (s *stubService) CreateActivityLogs(ctx context.Context, params repositories.CreateActivityLogParams) (*models.ActivityLog, error) {
	activity := createActivityModelFromParams(params)
	s.mu.Lock()
	s.created = append(s.created, activity)
	s.mu.Unlock()
	if s.createCh != nil {
		s.createCh <- struct{}{}
	}
	if s.createErr != nil {
		return nil, s.createErr
	}
	return activity, nil
}

func (s *stubService) UpdateActivityLogSessionID(ctx context.Context, params repositories.UpdateActivityLogSessionParams) (*models.ActivityLog, error) {
	activity := updateActivityModelFromParams(params)
	s.mu.Lock()
	s.updated = append(s.updated, activity)
	s.mu.Unlock()
	if s.updateCh != nil {
		s.updateCh <- struct{}{}
	}
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return activity, nil
}

func (s *stubService) GetActivityLogs(ctx context.Context, filter repositories.ActivityLogFilters) ([]models.ActivityLog, int64, error) {
	return nil, 0, nil
}

func (s *stubService) GetEventCategories(ctx context.Context) ([]string, error) {
	return nil, nil
}

func setClientService(t *testing.T, client *useractivity.Client, svc services.ActivityLogService) {
	t.Helper()
	v := reflect.ValueOf(client).Elem().FieldByName("svc")
	if !v.IsValid() {
		t.Fatalf("client.svc not found")
	}
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(svc))
}

func newTestClient(t *testing.T, svc services.ActivityLogService) *useractivity.Client {
	t.Helper()
	client := &useractivity.Client{}
	setClientService(t, client, svc)
	return client
}

func TestMiddlewareCaptureAndEnrich(t *testing.T) {
	svc := &stubService{}
	client := newTestClient(t, svc)
	var contextSessionID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sessionID, ok := useractivity.SessionIDFromContext(r.Context()); ok {
			contextSessionID = sessionID
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("ok"))
	})

	mw := Middleware(Config{
		Client:              client,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodyBytes:        1024,
		SessionIDHeader:     "X-Session-ID",
		CreateEnricher: func(r *http.Request, req *useractivity.CreateRequest) {
			req.APIStatus = useractivity.APIStatusSuccess
		},
		UpdateEnricher: func(r *http.Request, req *useractivity.UpdateRequest, resp *CapturedResponse) {
			if resp != nil && resp.StatusCode == http.StatusAccepted {
				msg := "accepted"
				req.Description = &msg
			}
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("payload"))
	req.Header.Set("X-Session-ID", "sess-1")
	req.Header.Set("X-Forwarded-For", "9.9.9.9")

	mw(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rec.Code)
	}
	if len(svc.created) != 1 || len(svc.updated) != 1 {
		t.Fatalf("expected create/update calls, got %d/%d", len(svc.created), len(svc.updated))
	}
	if contextSessionID != "sess-1" {
		t.Fatalf("expected session ID in request context")
	}
	if svc.created[0].IPAddress == nil || *svc.created[0].IPAddress != "9.9.9.9" {
		t.Fatalf("expected IP from X-Forwarded-For")
	}
	if svc.updated[0].ResponseBody == nil || *svc.updated[0].ResponseBody != "ok" {
		t.Fatalf("expected response body captured")
	}
}

func TestMiddlewareResponseRedact(t *testing.T) {
	svc := &stubService{}
	client := newTestClient(t, svc)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"secret","ok":true}`))
	})

	mw := Middleware(Config{
		Client:              client,
		CaptureResponseBody: true,
		ResponseRedact:      useractivity.RedactDefaultJSONKeys,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)

	if len(svc.updated) != 1 || svc.updated[0].ResponseBody == nil {
		t.Fatalf("expected response body to be captured")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(*svc.updated[0].ResponseBody), &payload); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if payload["token"] != "***REDACTED***" {
		t.Fatalf("expected token to be redacted, got %v", payload["token"])
	}
}

func TestMiddlewareSkipAndError(t *testing.T) {
	svc := &stubService{createErr: errors.New("create")}
	client := newTestClient(t, svc)
	var capturedErr error

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(Config{
		Client:    client,
		SkipPaths: []string{"/skip"},
		OnError:   func(err error) { capturedErr = err },
	})

	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)
	if len(svc.created) != 0 {
		t.Fatalf("expected no create for skipped path")
	}

	req = httptest.NewRequest(http.MethodGet, "/err", nil)
	rec = httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)
	if capturedErr == nil {
		t.Fatalf("expected error handler to be called")
	}
	if len(svc.updated) != 0 {
		t.Fatalf("expected update not called when create fails")
	}
}

func TestDefaultIPExtractor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "8.8.8.8")
	if ip := DefaultIPExtractor(req); ip != "8.8.8.8" {
		t.Fatalf("expected X-Real-IP, got %s", ip)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	if ip := DefaultIPExtractor(req); ip != "1.2.3.4" {
		t.Fatalf("expected parsed remote addr, got %s", ip)
	}
}

func TestDefaultIPExtractorSkipsEmptyForwardedForEntries(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "   ,   ,")
	req.Header.Set("X-Real-IP", "8.8.4.4")

	if ip := DefaultIPExtractor(req); ip != "8.8.4.4" {
		t.Fatalf("expected fallback to X-Real-IP, got %q", ip)
	}
}

func TestMiddlewareAsync(t *testing.T) {
	svc := &stubService{createCh: make(chan struct{}, 1), updateCh: make(chan struct{}, 1)}
	client := newTestClient(t, svc)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(Config{Client: client, Async: true})
	req := httptest.NewRequest(http.MethodGet, "/async", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)

	select {
	case <-svc.createCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected async create call")
	}

	select {
	case <-svc.updateCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected async update call")
	}
}

func TestMiddlewareGeoLookup(t *testing.T) {
	geoClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonRoundTripResponse(`{
			"country":"United States",
			"country_code":"US",
			"region":"California",
			"city":"Mountain View",
			"timezone":"America/Los_Angeles",
			"latitude":37.3861,
			"longitude":-122.0839
		}`), nil
		}),
	}

	lookup := useractivity.NewGeoLookup(useractivity.GeoLookupConfig{
		ProviderURLTemplate: "https://geo.local/json/%s",
		HTTPClient:          geoClient,
	})

	svc := &stubService{}
	client := newTestClient(t, svc)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(Config{
		Client:    client,
		GeoLookup: lookup,
		IPExtractor: func(*http.Request) string {
			return "8.8.8.8"
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/geo", nil)
	rec := httptest.NewRecorder()
	mw(handler).ServeHTTP(rec, req)

	if len(svc.created) != 1 {
		t.Fatalf("expected create call")
	}
	created := svc.created[0]
	if created.Country == nil || *created.Country != "United States" {
		t.Fatalf("expected geolocation country to be populated")
	}
	if created.Timezone == nil || *created.Timezone != "America/Los_Angeles" {
		t.Fatalf("expected geolocation timezone to be populated")
	}
}
