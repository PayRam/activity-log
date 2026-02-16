package useractivity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PayRam/user-activity-go/internal/models"
	"github.com/PayRam/user-activity-go/internal/repositories"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubService struct {
	createFn     func(*models.UserActivity) (*models.UserActivity, error)
	updateFn     func(*models.UserActivity) (*models.UserActivity, error)
	getFn        func(repositories.UserActivityFilters) ([]models.UserActivity, int64, error)
	categoriesFn func() ([]string, error)
	lastFilter   repositories.UserActivityFilters
	lastCreate   *models.UserActivity
	lastUpdate   *models.UserActivity
}

func (s *stubService) Create(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	s.lastCreate = activity
	if s.createFn != nil {
		return s.createFn(activity)
	}
	return activity, nil
}

func (s *stubService) UpdateBySessionID(ctx context.Context, activity *models.UserActivity) (*models.UserActivity, error) {
	s.lastUpdate = activity
	if s.updateFn != nil {
		return s.updateFn(activity)
	}
	return activity, nil
}

func (s *stubService) Get(ctx context.Context, filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
	s.lastFilter = filter
	if s.getFn != nil {
		return s.getFn(filter)
	}
	return nil, 0, nil
}

func (s *stubService) GetEventCategories(ctx context.Context) ([]string, error) {
	if s.categoriesFn != nil {
		return s.categoriesFn()
	}
	return nil, nil
}

type stubAccessResolver struct {
	access *AccessContext
	err    error
}

func (r *stubAccessResolver) Resolve(ctx context.Context, memberID uint) (*AccessContext, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.access, nil
}

type stubConfigProvider struct {
	val int
	ok  bool
	err error
}

func (p *stubConfigProvider) GetInt(ctx context.Context, key string) (int, bool, error) {
	return p.val, p.ok, p.err
}

type stubMemberResolver struct {
	data map[uint]MemberInfo
	err  error
}

func (r *stubMemberResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]MemberInfo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.data, nil
}

type stubPlatformResolver struct {
	data map[uint]ExternalPlatformInfo
	err  error
}

func (r *stubPlatformResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]ExternalPlatformInfo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.data, nil
}

func TestNewRequiresDB(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatalf("expected error for nil db")
	}
}

func TestNewSetsTablePrefix(t *testing.T) {
	models.ResetTablePrefix()
	t.Cleanup(models.ResetTablePrefix)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	_, err = New(Config{DB: db, TablePrefix: "ua_"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := models.GetTableName(models.DefaultUserActivityTableName); got != "ua_user_activities" {
		t.Fatalf("expected table name ua_user_activities, got %q", got)
	}
}

func TestNewSetsCustomTableName(t *testing.T) {
	models.ResetTablePrefix()
	t.Cleanup(models.ResetTablePrefix)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	_, err = New(Config{
		DB:          db,
		TablePrefix: "ua_",
		TableName:   "activity_logs",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := models.GetTableName(models.DefaultUserActivityTableName); got != "ua_activity_logs" {
		t.Fatalf("expected table name ua_activity_logs, got %q", got)
	}
}

func TestAutoMigrate(t *testing.T) {
	var client *Client
	if err := client.AutoMigrate(context.Background()); err == nil {
		t.Fatalf("expected error for nil client")
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	c, err := New(Config{DB: db})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := c.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auto-migrate error: %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	c := &Client{svc: &stubService{}}

	if _, err := c.Create(context.Background(), CreateRequest{}); err == nil {
		t.Fatalf("expected error for missing session_id")
	}

	req := CreateRequest{SessionID: "s"}
	if _, err := c.Create(context.Background(), req); err == nil {
		t.Fatalf("expected error for missing method")
	}
	req.Method = "GET"
	if _, err := c.Create(context.Background(), req); err == nil {
		t.Fatalf("expected error for missing endpoint")
	}
	req.Endpoint = "/x"
	if _, err := c.Create(context.Background(), req); err == nil {
		t.Fatalf("expected error for missing api_action")
	}
	req.APIAction = "READ"
	if _, err := c.Create(context.Background(), req); err == nil {
		t.Fatalf("expected error for missing api_status")
	}
}

func TestCreateMapping(t *testing.T) {
	stub := &stubService{
		createFn: func(activity *models.UserActivity) (*models.UserActivity, error) {
			activity.ID = 99
			return activity, nil
		},
	}
	c := &Client{svc: stub}

	memberID := uint(7)
	req := CreateRequest{
		MemberID:            &memberID,
		SessionID:           "sess",
		ExternalPlatformIDs: []uint{1, 2},
		Method:              "POST",
		Endpoint:            "/test",
		APIAction:           "WRITE",
		APIStatus:           "SUCCESS",
	}
	act, err := c.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if act == nil || act.ID != 99 || act.SessionID != "sess" {
		t.Fatalf("unexpected activity: %#v", act)
	}
	if stub.lastCreate == nil || stub.lastCreate.APIPart != "/test" {
		t.Fatalf("expected create mapping to set APIPart")
	}
}

func TestUpdateValidationAndMapping(t *testing.T) {
	c := &Client{svc: &stubService{}}
	if _, err := c.Update(context.Background(), UpdateRequest{}); err == nil {
		t.Fatalf("expected error for missing session_id")
	}

	stub := &stubService{
		updateFn: func(activity *models.UserActivity) (*models.UserActivity, error) {
			return activity, nil
		},
	}
	c = &Client{svc: stub}

	method := "PUT"
	endpoint := "/update"
	apiStatus := "SUCCESS"
	apiAction := "WRITE"
	req := UpdateRequest{
		SessionID:           "sess",
		ExternalPlatformIDs: []uint{9},
		Method:              &method,
		Endpoint:            &endpoint,
		APIStatus:           &apiStatus,
		APIAction:           &apiAction,
	}
	if _, err := c.Update(context.Background(), req); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if stub.lastUpdate == nil || stub.lastUpdate.Method != method {
		t.Fatalf("expected update to map optional fields")
	}
}

func TestGetAccessScopeAndExportLimit(t *testing.T) {
	stub := &stubService{
		getFn: func(filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
			return nil, 0, nil
		},
	}

	c := &Client{
		svc:            stub,
		accessResolver: &stubAccessResolver{access: &AccessContext{IsAdmin: false, AllowedProjectIDs: []uint{1, 2}}},
		configProvider: &stubConfigProvider{val: 10, ok: true},
	}

	projectFilter := "ALL"
	req := GetRequest{ProjectFilter: &projectFilter, Export: true}
	if _, err := c.Get(context.Background(), 5, req); err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if stub.lastFilter.Limit == nil || *stub.lastFilter.Limit != 10 {
		t.Fatalf("expected limit capped to export limit")
	}
	if len(stub.lastFilter.ExternalPlatformIDs) != 2 {
		t.Fatalf("expected access scope to set external platform IDs")
	}

	req = GetRequest{ExternalPlatformIDs: []uint{3}}
	if _, err := c.Get(context.Background(), 5, req); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestGetDateHandlingAndResolvers(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	stub := &stubService{
		getFn: func(filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
			return []models.UserActivity{
				{
					BaseModel:           models.BaseModel{ID: 1},
					SessionID:           "s1",
					ExternalPlatformIDs: models.UintSlice{2},
					MemberID:            uintPtr(5),
				},
			}, 1, nil
		},
	}

	c := &Client{
		svc:                      stub,
		memberResolver:           &stubMemberResolver{data: map[uint]MemberInfo{5: {ID: 5, Name: "A"}}},
		externalPlatformResolver: &stubPlatformResolver{data: map[uint]ExternalPlatformInfo{2: {ID: 2, Name: "P"}}},
	}

	req := GetRequest{PaginationConditions: PaginationConditions{StartDate: &start, EndDate: &future}}
	resp, err := c.Get(context.Background(), 0, req)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if stub.lastFilter.EndDate == nil || stub.lastFilter.EndDate.After(time.Now()) {
		t.Fatalf("expected end date capped to now")
	}
	if len(resp.Activities) != 1 || resp.Activities[0].Member == nil {
		t.Fatalf("expected member resolver to populate member info")
	}
	if len(resp.Activities[0].ExternalPlatforms) != 1 {
		t.Fatalf("expected external platform resolver to populate platform info")
	}
}

func TestGetResolverErrors(t *testing.T) {
	stub := &stubService{
		getFn: func(filter repositories.UserActivityFilters) ([]models.UserActivity, int64, error) {
			return []models.UserActivity{{MemberID: uintPtr(1)}}, 1, nil
		},
	}
	c := &Client{
		svc:            stub,
		memberResolver: &stubMemberResolver{err: errors.New("resolver")},
	}

	if _, err := c.Get(context.Background(), 0, GetRequest{}); err == nil {
		t.Fatalf("expected resolver error")
	}
}

func TestGetEventCategories(t *testing.T) {
	stub := &stubService{
		categoriesFn: func() ([]string, error) {
			return []string{"AUTH"}, nil
		},
	}
	c := &Client{svc: stub}

	cats, err := c.GetEventCategories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 1 || cats[0] != "AUTH" {
		t.Fatalf("unexpected categories: %v", cats)
	}
}

func uintPtr(v uint) *uint {
	return &v
}
