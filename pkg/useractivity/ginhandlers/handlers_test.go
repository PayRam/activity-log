package ginhandlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/PayRam/user-activity-go/internal/models"
	"github.com/PayRam/user-activity-go/internal/repositories"
	"github.com/PayRam/user-activity-go/internal/services"
	"github.com/PayRam/user-activity-go/pkg/useractivity"
	"github.com/gin-gonic/gin"
)

type stubService struct {
	listItems []models.UserActivity
	total     int64
	cats      []string
	listErr   error
	catErr    error
}

func (s *stubService) Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	return activity, nil
}

func (s *stubService) UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	return activity, nil
}

func (s *stubService) Get(ctx context.Context, filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listItems, s.total, nil
}

func (s *stubService) GetEventCategories(ctx context.Context) ([]string, error) {
	if s.catErr != nil {
		return nil, s.catErr
	}
	return s.cats, nil
}

func setClientService(t *testing.T, client *useractivity.Client, svc services.UserActivityService) {
	t.Helper()
	v := reflect.ValueOf(client).Elem().FieldByName("svc")
	if !v.IsValid() {
		t.Fatalf("client.svc not found")
	}
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(svc))
}

func newTestClient(t *testing.T, svc services.UserActivityService) *useractivity.Client {
	t.Helper()
	client := &useractivity.Client{}
	setClientService(t, client, svc)
	return client
}

func TestGetUserActivities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{
		listItems: []models.UserActivity{
			{
				BaseModel: models.BaseModel{ID: 1},
				SessionID: "sess",
				Method:    "GET",
				APIPart:   "/test",
				APIStatus: "SUCCESS",
				APIAction: "READ",
			},
		},
		total: 1,
	}

	client := newTestClient(t, svc)
	handler := NewHandler(HandlerConfig{
		Client: client,
		MemberIDFromContext: func(c *gin.Context) (uint, bool) {
			return 1, true
		},
		RequireMember: true,
	})

	router := gin.New()
	SetupRoutes(router.Group(""), handler)

	req := httptest.NewRequest(http.MethodGet, "/user-activity", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if payload["total_count"].(float64) != 1 {
		t.Fatalf("expected total_count 1, got %v", payload["total_count"])
	}
	data := payload["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 data row, got %d", len(data))
	}
	row := data[0].(map[string]interface{})
	if row["session_id"] != "sess" {
		t.Fatalf("expected session_id sess, got %v", row["session_id"])
	}
}

func TestGetUserActivitiesUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newTestClient(t, &stubService{})
	handler := NewHandler(HandlerConfig{
		Client: client,
		MemberIDFromContext: func(c *gin.Context) (uint, bool) {
			return 0, false
		},
		RequireMember: true,
	})

	router := gin.New()
	SetupRoutes(router.Group(""), handler)

	req := httptest.NewRequest(http.MethodGet, "/user-activity", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGetUserActivitiesBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := newTestClient(t, &stubService{})
	handler := NewHandler(HandlerConfig{
		Client: client,
		MemberIDFromContext: func(c *gin.Context) (uint, bool) {
			return 1, true
		},
		RequireMember: true,
	})

	router := gin.New()
	SetupRoutes(router.Group(""), handler)

	req := httptest.NewRequest(http.MethodGet, "/user-activity?externalPlatformIDs=1&projectFilter=ALL", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDownloadUserActivitiesCSV(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{
		listItems: []models.UserActivity{
			{
				BaseModel: models.BaseModel{ID: 2},
				SessionID: "s2",
				Method:    "POST",
				APIPart:   "/export",
				APIStatus: "SUCCESS",
				APIAction: "WRITE",
			},
		},
		total: 1,
	}
	client := newTestClient(t, svc)
	handler := NewHandler(HandlerConfig{
		Client: client,
		MemberIDFromContext: func(c *gin.Context) (uint, bool) {
			return 1, true
		},
		RequireMember: true,
	})

	router := gin.New()
	SetupRoutes(router.Group(""), handler)

	req := httptest.NewRequest(http.MethodGet, "/user-activity/export", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("expected text/csv content type")
	}
	if !strings.Contains(rec.Body.String(), "session_id") {
		t.Fatalf("expected csv header")
	}
	if !strings.Contains(rec.Body.String(), "s2") {
		t.Fatalf("expected csv row with session id")
	}
}

func TestGetEventCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubService{cats: []string{"AUTH", "PAYMENT"}}
	client := newTestClient(t, svc)
	handler := NewHandler(HandlerConfig{
		Client: client,
		MemberIDFromContext: func(c *gin.Context) (uint, bool) {
			return 1, true
		},
		RequireMember: true,
	})

	router := gin.New()
	SetupRoutes(router.Group(""), handler)

	req := httptest.NewRequest(http.MethodGet, "/user-activity/event-categories", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "AUTH") {
		t.Fatalf("expected categories in response")
	}
}
