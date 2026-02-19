package ginmiddleware

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
	activitylog "github.com/PayRam/activity-log/pkg/activitylog"
	"github.com/gin-gonic/gin"
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

func setClientService(t *testing.T, client *activitylog.Client, svc services.ActivityLogService) {
	t.Helper()
	v := reflect.ValueOf(client).Elem().FieldByName("svc")
	if !v.IsValid() {
		t.Fatalf("client.svc not found")
	}
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(svc))
}

func newTestClient(t *testing.T, svc services.ActivityLogService) *activitylog.Client {
	t.Helper()
	client := &activitylog.Client{}
	setClientService(t, client, svc)
	return client
}

func TestMiddlewareCaptureAndEnrich(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{}
	client := newTestClient(t, svc)
	var contextSessionID string

	router := gin.New()
	router.Use(Middleware(Config{
		Client:              client,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodyBytes:        1024,
		SessionIDHeader:     "X-Session-ID",
		IPExtractor: func(c *gin.Context) string {
			return "1.2.3.4"
		},
		CreateEnricher: func(c *gin.Context, req *activitylog.CreateRequest) {
			req.APIStatus = activitylog.APIStatusSuccess
		},
		UpdateEnricher: func(c *gin.Context, req *activitylog.UpdateRequest, resp *CapturedResponse) {
			if resp != nil && resp.StatusCode == http.StatusCreated {
				msg := "created"
				req.Description = &msg
			}
		},
	}))

	router.POST("/test", func(c *gin.Context) {
		if sessionID, ok := activitylog.SessionIDFromContext(c.Request.Context()); ok {
			contextSessionID = sessionID
		}
		c.String(http.StatusCreated, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("payload"))
	req.Header.Set("X-Session-ID", "sess-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	if len(svc.created) != 1 || len(svc.updated) != 1 {
		t.Fatalf("expected create/update calls, got %d/%d", len(svc.created), len(svc.updated))
	}
	created := svc.created[0]
	if created.SessionID != "sess-1" || created.Method != http.MethodPost {
		t.Fatalf("unexpected create payload: %#v", created)
	}
	if contextSessionID != "sess-1" {
		t.Fatalf("expected session ID in request context")
	}
	if created.IPAddress == nil || *created.IPAddress != "1.2.3.4" {
		t.Fatalf("expected IP address to be set")
	}
	if created.RequestBody == nil || *created.RequestBody != "payload" {
		t.Fatalf("expected request body to be captured")
	}

	updated := svc.updated[0]
	if updated.StatusCode == nil || *updated.StatusCode != http.StatusCreated {
		t.Fatalf("expected update status code to be set")
	}
	if updated.ResponseBody == nil || *updated.ResponseBody != "ok" {
		t.Fatalf("expected response body to be captured")
	}
	if updated.Description == nil || *updated.Description != "created" {
		t.Fatalf("expected update enricher to set description")
	}
}

func TestMiddlewareUsesDefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ResetDefaultConfig()
	t.Cleanup(ResetDefaultConfig)

	svc := &stubService{}
	client := newTestClient(t, svc)
	SetDefaultConfig(Config{Client: client})

	router := gin.New()
	router.Use(Middleware())
	router.GET("/default", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/default", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if len(svc.created) != 1 || len(svc.updated) != 1 {
		t.Fatalf("expected create/update calls via default config, got %d/%d", len(svc.created), len(svc.updated))
	}
}

func TestMiddlewareResponseRedact(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{}
	client := newTestClient(t, svc)

	router := gin.New()
	router.Use(Middleware(Config{
		Client:              client,
		CaptureResponseBody: true,
		ResponseRedact:      activitylog.RedactDefaultJSONKeys,
	}))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "secret", "ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

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
	gin.SetMode(gin.TestMode)
	svc := &stubService{createErr: errors.New("create")}
	client := newTestClient(t, svc)
	var capturedErr error

	router := gin.New()
	router.Use(Middleware(Config{
		Client:    client,
		SkipPaths: []string{"/skip"},
		OnError:   func(err error) { capturedErr = err },
	}))
	router.GET("/skip", func(c *gin.Context) { c.String(http.StatusOK, "skip") })
	router.GET("/err", func(c *gin.Context) { c.String(http.StatusOK, "err") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	router.ServeHTTP(rec, req)
	if len(svc.created) != 0 {
		t.Fatalf("expected no create for skipped path")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/err", nil)
	router.ServeHTTP(rec, req)
	if capturedErr == nil {
		t.Fatalf("expected error handler to be called")
	}
	if len(svc.updated) != 0 {
		t.Fatalf("expected update not called when create fails")
	}
}

func TestMiddlewareAsync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{createCh: make(chan struct{}, 1), updateCh: make(chan struct{}, 1)}
	client := newTestClient(t, svc)

	router := gin.New()
	router.Use(Middleware(Config{
		Client: client,
		Async:  true,
	}))
	router.GET("/async", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/async", nil)
	router.ServeHTTP(rec, req)

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
	gin.SetMode(gin.TestMode)

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

	lookup := activitylog.NewGeoLookup(activitylog.GeoLookupConfig{
		ProviderURLTemplate: "https://geo.local/json/%s",
		HTTPClient:          geoClient,
	})

	svc := &stubService{}
	client := newTestClient(t, svc)

	router := gin.New()
	router.Use(Middleware(Config{
		Client:    client,
		GeoLookup: lookup,
		IPExtractor: func(*gin.Context) string {
			return "8.8.8.8"
		},
	}))
	router.GET("/geo", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/geo", nil)
	router.ServeHTTP(rec, req)

	if len(svc.created) != 1 {
		t.Fatalf("expected create call")
	}
	created := svc.created[0]
	if created.Country == nil || *created.Country != "United States" {
		t.Fatalf("expected geolocation country to be populated")
	}
	if created.City == nil || *created.City != "Mountain View" {
		t.Fatalf("expected geolocation city to be populated")
	}
}
