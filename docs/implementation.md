# Technical Implementation

This document explains how the library works internally and gives concrete integration examples for every major feature.

## 1) Bootstrapping the Client

Main entrypoint: `pkg/useractivity/activity_log.go`.

Key behaviors:

- `DB` is required.
- logger defaults to production zap when not provided.
- table settings are applied at startup (`TablePrefix`, `TableName`).
- repository/service are wired automatically.

### Sample: full client setup

```go
package main

import (
	"context"

	"github.com/PayRam/activity-log/pkg/useractivity"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type accessResolver struct{}

type configProvider struct{}

type memberResolver struct{}

type projectResolver struct{}

func (r *accessResolver) Resolve(ctx context.Context, memberID uint) (*useractivity.AccessContext, error) {
	return &useractivity.AccessContext{IsAdmin: false, AllowedProjectIDs: []uint{101, 102}}, nil
}

func (p *configProvider) GetInt(ctx context.Context, key string) (int, bool, error) {
	if key == useractivity.ConfigKeyActivityLogExportLimit {
		return 2000, true, nil
	}
	return 0, false, nil
}

func (r *memberResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]useractivity.MemberInfo, error) {
	out := map[uint]useractivity.MemberInfo{}
	for _, id := range ids {
		out[id] = useractivity.MemberInfo{ID: id, Name: "member"}
	}
	return out, nil
}

func (r *projectResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]useractivity.ProjectInfo, error) {
	out := map[uint]useractivity.ProjectInfo{}
	for _, id := range ids {
		out[id] = useractivity.ProjectInfo{ID: id, Name: "project"}
	}
	return out, nil
}

func newClient() (*useractivity.Client, error) {
	db, err := gorm.Open(postgres.Open("your-dsn"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	logger, _ := zap.NewProduction()

	client, err := useractivity.New(useractivity.Config{
		DB:          db,
		Logger:      logger,
		TablePrefix: "core_",
		TableName:   "activity_logs",
		EventInfoDeriver: useractivity.NewCoreLikeEventInfoDeriver(useractivity.CoreLikeEventDeriverConfig{
			BasePath: "/api/v1",
		}),
		AccessResolver:  &accessResolver{},
		ConfigProvider:  &configProvider{},
		MemberResolver:  &memberResolver{},
		ProjectResolver: &projectResolver{},
	})
	if err != nil {
		return nil, err
	}

	if err := client.AutoMigrate(context.Background()); err != nil {
		return nil, err
	}
	return client, nil
}
```

## 2) Create Flow (`CreateActivityLogs`)

What happens internally:

1. validates required fields (`SessionID`, `Method`, `Endpoint`, `APIAction`, `APIStatus`).
2. derives missing event fields if event derivers are configured.
3. maps public request -> repository params.
4. persists through service/repository.

### Sample: create a log

```go
package main

import (
	"context"
	"net/http"

	"github.com/PayRam/activity-log/pkg/useractivity"
)

func createExample(client *useractivity.Client) error {
	status := useractivity.HTTPStatusCode(http.StatusCreated)
	desc := "payment request created"

	_, err := client.CreateActivityLogs(context.Background(), useractivity.CreateRequest{
		SessionID:  "sess-123",
		MemberID:   uintPtr(77),
		ProjectIDs: []uint{101},
		Method:     http.MethodPost,
		Endpoint:   "/api/v1/payment-request",
		APIAction:  useractivity.APIActionWrite,
		APIStatus:  useractivity.APIStatusSuccess,
		StatusCode: &status,
		Description: &desc,
	})
	return err
}

func uintPtr(v uint) *uint { return &v }
```

## 3) Update Flow (`UpdateActivityLogSessionID`)

What happens internally:

1. validates `SessionID`.
2. maps optional fields into update params.
3. repository transaction + row lock by `session_id`.
4. update map only includes fields explicitly set.

`ProjectIDs` update semantics:

- `nil pointer` => no change.
- `&nilSlice` => set DB JSON column to `NULL`.
- `&[]uint{}` => set DB JSON column to `[]`.
- `&[]uint{1,2}` => set DB JSON column to `[1,2]`.

### Sample: update API status and metadata

```go
package main

import (
	"context"
	"net/http"

	"github.com/PayRam/activity-log/pkg/useractivity"
)

func updateExample(client *useractivity.Client) error {
	status := useractivity.APIStatusDenied
	code := useractivity.HTTPStatusCode(http.StatusForbidden)
	msg := "permission denied"

	// set project IDs to NULL in DB
	var nilProjects []uint

	_, err := client.UpdateActivityLogSessionID(context.Background(), useractivity.UpdateRequest{
		SessionID:   "sess-123",
		APIStatus:   &status,
		StatusCode:  &code,
		Description: &msg,
		ProjectIDs:  &nilProjects,
	})
	return err
}
```

## 4) Get Flow (`GetActivityLogs`)

What happens internally:

- validates request combinations.
- applies access scope from `AccessResolver`.
- builds filter query.
- applies default/export limits.
- optionally hydrates member/platform maps.

### Sample: get logs with filters

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/PayRam/activity-log/pkg/useractivity"
)

func listExample(client *useractivity.Client) (useractivity.GetResponse, error) {
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	limit := 50
	offset := 0
	sortBy := "created_at"
	order := "DESC"

	pf := useractivity.ProjectFilterAll

	return client.GetActivityLogs(context.Background(), 77, useractivity.GetRequest{
		StatusCodes: []useractivity.HTTPStatusCode{
			useractivity.HTTPStatusCode(http.StatusOK),
			useractivity.HTTPStatusCode(http.StatusBadRequest),
		},
		Methods:       []string{http.MethodGet, http.MethodPost},
		EventNames:    []string{"PAYMENT_CREATE", "WITHDRAWAL_APPROVE"},
		ProjectFilter: &pf,
		PaginationConditions: useractivity.PaginationConditions{
			Limit:     &limit,
			Offset:    &offset,
			SortBy:    &sortBy,
			Order:     &order,
			StartDate: &from,
			EndDate:   &to,
		},
	})
}
```

## 5) Middleware Implementations

Files:

- Gin: `pkg/useractivity/ginmiddleware/middleware.go`
- net/http: `pkg/useractivity/httpmiddleware/middleware.go`

Lifecycle:

1. create/reuse session ID.
2. optional request body capture + redaction.
3. call `CreateActivityLogs` at request start.
4. capture response status/body.
5. call `UpdateActivityLogSessionID` at response end.

### Sample: Gin middleware

```go
package main

import (
	"net/http"

	"github.com/PayRam/activity-log/pkg/useractivity"
	"github.com/PayRam/activity-log/pkg/useractivity/ginmiddleware"
	"github.com/gin-gonic/gin"
)

func wireGin(r *gin.Engine, client *useractivity.Client) {
	r.Use(ginmiddleware.Middleware(ginmiddleware.Config{
		Client:              client,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodyBytes:        1 * 1024 * 1024,
		Redact:              useractivity.RedactDefaultJSONKeys,
		ResponseRedact:      useractivity.RedactDefaultJSONKeys,
		SkipPaths:           []string{"/signin", "/signup", "/oauth/token"},
		CreateEnricher: func(c *gin.Context, req *useractivity.CreateRequest) {
			memberID := uint(77)
			req.MemberID = &memberID
			req.ProjectIDs = []uint{101}
		},
		UpdateEnricher: func(c *gin.Context, req *useractivity.UpdateRequest, resp *ginmiddleware.CapturedResponse) {
			if resp.StatusCode >= http.StatusBadRequest && req.Description == nil {
				msg := "request failed"
				req.Description = &msg
			}
		},
	}))
}
```

### Sample: net/http middleware

```go
package main

import (
	"net/http"

	"github.com/PayRam/activity-log/pkg/useractivity"
	"github.com/PayRam/activity-log/pkg/useractivity/httpmiddleware"
)

func wrapHTTP(next http.Handler, client *useractivity.Client) http.Handler {
	mw := httpmiddleware.Middleware(httpmiddleware.Config{
		Client:              client,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		MaxBodyBytes:        1 * 1024 * 1024,
		SkipPaths:           []string{"/signin", "/signup"},
		Redact:              useractivity.RedactDefaultJSONKeys,
		ResponseRedact:      useractivity.RedactDefaultJSONKeys,
	})
	return mw(next)
}
```

## 6) Service Tracker (`ServiceTracker.Track`)

File: `pkg/useractivity/service_tracker.go`.

Behavior:

1. ensures function exists.
2. creates/reuses session ID.
3. writes "STARTED" operation metadata.
4. runs operation.
5. converts error -> `APIStatus`.
6. writes final operation trail status.

### Sample: track service operation

```go
package main

import (
	"context"

	"github.com/PayRam/activity-log/pkg/useractivity"
)

func trackedOperation(ctx context.Context, client *useractivity.Client) error {
	tracker := useractivity.NewServiceTracker(useractivity.ServiceTrackerConfig{Client: client})
	memberID := uint(77)
	category := "PAYMENT"
	name := "CREATE_PAYMENT_REQUEST"

	return tracker.Track(ctx, useractivity.ServiceOperation{
		Name:          "PaymentService.CreateNewPaymentRequest",
		MemberID:      &memberID,
		ProjectIDs:    []uint{101},
		APIAction:     useractivity.APIActionWrite,
		EventCategory: &category,
		EventName:     &name,
	}, func(ctx context.Context) error {
		// do repository work here
		return nil
	})
}
```

## 7) Event Derivation

File: `pkg/useractivity/event_deriver.go`.

You can use built-in derivation or provide your own function.

### Sample: custom event info deriver

```go
client, err := useractivity.New(useractivity.Config{
	DB: db,
	EventInfoDeriver: func(input useractivity.EventDeriverInput) useractivity.EventInfo {
		if input.Method == "POST" && input.Endpoint == "/api/v1/payment-request" {
			return useractivity.EventInfo{
				EventCategory: "PAYMENT",
				EventName:     "CREATE_PAYMENT_REQUEST",
				Description:   "created payment request",
			}
		}
		return useractivity.DefaultEventInfoDeriver(input)
	},
})
```

## 8) Geolocation

File: `pkg/useractivity/geolocation.go`.

### Sample: lookup + enrich request

```go
lookup := useractivity.NewGeoLookup(useractivity.GeoLookupConfig{
	ProviderURLTemplate: "https://ipwhois.app/json/%s",
	ProviderName:        "ipwhois.io",
	Timeout:             5 * time.Second,
	CacheTTL:            24 * time.Hour,
})

loc := lookup.Lookup("8.8.8.8")
if loc != nil {
	useractivity.EnrichCreateRequestWithLocation(&createReq, loc)
	useractivity.EnrichUpdateRequestWithLocation(&updateReq, loc)
}
```

Environment fallbacks when config values are empty:

- `GEOLOCATION_PROVIDER_URL`
- `GEOLOCATION_PROVIDER_NAME`

## 9) Metadata and Redaction

### Sample: merge metadata safely

```go
meta := useractivity.MergeMetadata(nil, map[string]any{
	"service": "PaymentService",
	"step":    "CreateNewPaymentRequest",
})

req := useractivity.UpdateRequest{SessionID: "sess-123", Metadata: meta}
```

### Sample: redact JSON payload

```go
raw := []byte(`{"email":"a@b.com","password":"secret","token":"jwt"}`)
redacted := useractivity.RedactDefaultJSONKeys(raw)
// password/token will be masked
```

## 10) Error Model

File: `pkg/useractivity/errors.go`.

- `ErrUnauthorized`
- `ErrBadRequest`

`ErrorToAPIStatus(err)` converts errors to `APIStatus`:

- `nil` -> `SUCCESS`
- `ErrUnauthorized` -> `DENIED`
- others -> `ERROR`

## 11) Security Guidelines

- Always skip auth and password/token endpoints in middleware.
- Use redaction when body capture is enabled.
- Avoid putting secrets into `Metadata`.
- Keep access scope strict via `AccessResolver`.
