# user-activity-go

Go library for persisting user activity logs with:

- direct APIs: `Create`, `Update`, `Get`, `GetEventCategories`
- optional HTTP middleware: Gin and `net/http`
- optional service-level tracker for business/repository operations
- metadata merge, JSON redaction, and geolocation helpers

Repository: `https://github.com/PayRam/activity-log`  
Module path: `github.com/PayRam/user-activity-go`

## Install

```bash
go get github.com/PayRam/user-activity-go
```

## Quick Start

```go
package main

import (
	"context"

	"github.com/PayRam/user-activity-go/pkg/useractivity"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func run() error {
	db, err := gorm.Open(postgres.Open("dsn"), &gorm.Config{})
	if err != nil {
		return err
	}

	logger, _ := zap.NewProduction()

	client, err := useractivity.New(useractivity.Config{
		DB:          db,
		Logger:      logger,
		TablePrefix: "",
	})
	if err != nil {
		return err
	}

	if err := client.AutoMigrate(context.Background()); err != nil {
		return err
	}

	_, err = client.Create(context.Background(), useractivity.CreateRequest{
		SessionID: "session-123",
		Method:    "POST",
		Endpoint:  "/api/v1/payment-request",
		APIAction: useractivity.APIActionWrite,
		APIStatus: useractivity.APIStatusSuccess,
	})
	if err != nil {
		return err
	}

	status := useractivity.APIStatusError
	code := 500
	msg := "downstream failed"
	_, err = client.Update(context.Background(), useractivity.UpdateRequest{
		SessionID:   "session-123",
		APIStatus:   &status,
		StatusCode:  &code,
		Description: &msg,
	})
	return err
}
```

## Architecture

Recommended production flow is hybrid:

1. middleware logs API request/response lifecycle
2. service tracker logs internal service/repository operations
3. shared `session_id` in context correlates both layers

This is useful when one API call triggers multiple service operations.

## File Structure

```text
user-activity-go/
├── cmd/
│   └── example/                     # runnable integration sample
├── pkg/
│   └── useractivity/                # public API surface
│       ├── user_activity.go         # Client + Create/Update/Get APIs
│       ├── types.go                 # request/response contracts
│       ├── service_tracker.go       # service/repository operation tracker
│       ├── geolocation.go           # public geolocation API + enrichers
│       ├── metadata.go              # public metadata merge helper
│       ├── redact.go                # public redaction helper
│       ├── ginmiddleware/           # Gin middleware
│       ├── httpmiddleware/          # net/http middleware
│       └── ginhandlers/             # optional read/export handlers
├── internal/                        # private implementation details
│   ├── models/                      # gorm models + table config
│   ├── repositories/                # gorm query + persistence layer
│   ├── services/                    # service layer used by Client
│   ├── middleware/                  # shared request/response capture utils
│   └── utils/                       # internal helpers (metadata/redact/ids/geolocation engine)
├── go.mod
└── README.md
```

Dependency direction:

- app imports `pkg/useractivity` (and optional `pkg/useractivity/ginmiddleware`, `pkg/useractivity/httpmiddleware`, `pkg/useractivity/ginhandlers`)
- `pkg/useractivity` uses `internal/services`
- `internal/services` uses `internal/repositories`
- `internal/repositories` uses `internal/models` + `gorm`
- `internal/*` packages are implementation details and not for external imports

## Client Config

`useractivity.Config`:

- `DB *gorm.DB` (required): gorm database handle
- `Logger *zap.Logger` (optional): defaults to production zap logger
- `TablePrefix string` (optional): prefixes `user_activities` table name
- `TableName string` (optional): overrides base table name (for example `activity_logs`)
- `AccessResolver AccessResolver` (optional): applies `Get` access scoping
- `ConfigProvider ConfigProvider` (optional): can override export limit (`user.activity.export.limit`)
- `MemberResolver MemberResolver` (optional): hydrates `Activity.Member` in `Get`
- `ExternalPlatformResolver ExternalPlatformResolver` (optional): hydrates `Activity.ExternalPlatforms` in `Get`

## Integrating with Different Naming

This library uses `ExternalPlatform` naming in the API, but your app can map it to any domain term (for example `Project`, `Tenant`, `Workspace`).

Common mapping:

| Library field | Your app (example) |
| --- | --- |
| `ExternalPlatformIDs` | `ProjectIDs` |
| `ExternalPlatformResolver` | `ProjectResolver` |
| `AccessContext.AllowedProjectIDs` | IDs of projects user can access |
| `ProjectFilter` | project-scope filter behavior |

Practical integration rule:

- Treat `ExternalPlatformIDs` as the scope IDs from your domain model.
- Keep your domain naming internally.
- Add a thin adapter only at integration boundaries.

Example adapter setup (`Project` -> `ExternalPlatform`):

```go
type ProjectService interface {
	GetByIDs(ctx context.Context, ids []uint) (map[uint]Project, error)
	GetAllowedProjectIDs(ctx context.Context, memberID uint) ([]uint, error)
}

type Project struct {
	ID       uint
	Name     string
	LogoPath string
}

type projectResolver struct {
	projectService ProjectService
}

func (r *projectResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]useractivity.ExternalPlatformInfo, error) {
	projects, err := r.projectService.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uint]useractivity.ExternalPlatformInfo, len(projects))
	for id, p := range projects {
		out[id] = useractivity.ExternalPlatformInfo{
			ID:       p.ID,
			Name:     p.Name,
			LogoPath: p.LogoPath,
		}
	}
	return out, nil
}

type projectAccessResolver struct {
	projectService ProjectService
}

func (r *projectAccessResolver) Resolve(ctx context.Context, memberID uint) (*useractivity.AccessContext, error) {
	allowed, err := r.projectService.GetAllowedProjectIDs(ctx, memberID)
	if err != nil {
		return nil, err
	}
	return &useractivity.AccessContext{
		IsAdmin:           false,
		AllowedProjectIDs: allowed,
	}, nil
}

client, err := useractivity.New(useractivity.Config{
	DB:                       db,
	ExternalPlatformResolver: &projectResolver{projectService: projectService},
	AccessResolver:           &projectAccessResolver{projectService: projectService},
})
```

Request mapping example:

```go
projectID := uint(101)
_, err := client.Create(ctx, useractivity.CreateRequest{
	SessionID:           "s-1",
	Method:              "POST",
	Endpoint:            "/payments",
	APIAction:           useractivity.APIActionWrite,
	APIStatus:           useractivity.APIStatusSuccess,
	ExternalPlatformIDs: []uint{projectID}, // project scope in your app
})
```

## API Reference

### `Create(ctx, CreateRequest)` (store)

Required fields:

- `SessionID string`
- `Method string`
- `Endpoint string`
- `APIAction string`
- `APIStatus string`

Supported optional fields:

- actor scope: `MemberID *uint`, `ExternalPlatformIDs []uint`
- result: `StatusCode *int`, `Description *string`, `APIErrorMsg *string`
- request info: `IPAddress *string`, `UserAgent *string`, `Referer *string`
- payloads: `RequestBody *string`, `ResponseBody *string`, `Metadata *string`
- classification: `Role *string`, `EventCategory *string`, `EventName *string`
- geolocation: `Country *string`, `CountryCode *string`, `Region *string`, `City *string`, `Timezone *string`, `Latitude *float64`, `Longitude *float64`

Notes:

- `Endpoint` is stored in DB field `api_part`.
- `ExternalPlatformIDs` is stored as JSON/JSONB array.

### `Update(ctx, UpdateRequest)`

Required field:

- `SessionID string`

Supported updatable fields:

- actor scope: `MemberID *uint`, `ExternalPlatformIDs []uint`
- route/action/status: `Method *string`, `Endpoint *string`, `APIAction *string`, `APIStatus *string`
- result: `StatusCode *int`, `Description *string`, `APIErrorMsg *string`
- request info: `IPAddress *string`, `UserAgent *string`, `Referer *string`
- payloads: `RequestBody *string`, `ResponseBody *string`, `Metadata *string`
- classification: `Role *string`, `EventCategory *string`, `EventName *string`
- geolocation: `Country *string`, `CountryCode *string`, `Region *string`, `City *string`, `Timezone *string`, `Latitude *float64`, `Longitude *float64`

### Store/Update parameter cheat sheet

Legend:

- `R`: required
- `O`: optional
- `-`: not applicable

Core fields:

| Field | Type | Create | Update | Notes |
| --- | --- | --- | --- | --- |
| `SessionID` | `string` | `R` | `R` | Update key (`session_id`) |
| `Method` | `string` / `*string` | `R` | `O` | HTTP method or service method |
| `Endpoint` | `string` / `*string` | `R` | `O` | Stored in DB as `api_part` |
| `APIAction` | `string` / `*string` | `R` | `O` | Use constants (`READ/WRITE/DELETE`) |
| `APIStatus` | `string` / `*string` | `R` | `O` | Use constants (`SUCCESS/DENIED/ERROR`) |
| `StatusCode` | `*int` | `O` | `O` | HTTP or domain status code |
| `Description` | `*string` | `O` | `O` | Human-readable message |
| `APIErrorMsg` | `*string` | `O` | `O` | Error text if any |

Actor and classification:

| Field | Type | Create | Update | Notes |
| --- | --- | --- | --- | --- |
| `MemberID` | `*uint` | `O` | `O` | Actor member id |
| `ExternalPlatformIDs` | `[]uint` | `O` | `O` | Project/platform scope |
| `Role` | `*string` | `O` | `O` | Actor role |
| `EventCategory` | `*string` | `O` | `O` | Event grouping |
| `EventName` | `*string` | `O` | `O` | Event name |

Request/response context:

| Field | Type | Create | Update | Notes |
| --- | --- | --- | --- | --- |
| `IPAddress` | `*string` | `O` | `O` | Client/source IP |
| `UserAgent` | `*string` | `O` | `O` | Caller user-agent |
| `Referer` | `*string` | `O` | `O` | HTTP referer |
| `RequestBody` | `*string` | `O` | `O` | Consider redaction |
| `ResponseBody` | `*string` | `O` | `O` | Consider redaction |
| `Metadata` | `*string` | `O` | `O` | JSON string recommended |

Geolocation:

| Field | Type | Create | Update | Notes |
| --- | --- | --- | --- | --- |
| `Country` | `*string` | `O` | `O` | Country name |
| `CountryCode` | `*string` | `O` | `O` | ISO-like code |
| `Region` | `*string` | `O` | `O` | Region/state |
| `City` | `*string` | `O` | `O` | City |
| `Timezone` | `*string` | `O` | `O` | Timezone |
| `Latitude` | `*float64` | `O` | `O` | Geo latitude |
| `Longitude` | `*float64` | `O` | `O` | Geo longitude |

Pointer semantics for update:

- `nil` pointer field: no change
- non-`nil` pointer field: update to provided value
- `ExternalPlatformIDs == nil`: no change
- `ExternalPlatformIDs != nil`: field is updated

Implementation detail:

- update is by `session_id`, inside a DB transaction with row lock
- GORM struct updates omit zero values for non-pointer fields

### `Get(ctx, memberID, GetRequest)`

Supported filters:

- exact/single: `StatusCode`, `Search`, `SessionID`, `ProjectFilter`
- arrays: `EventCategories`, `Methods`, `EventNames`, `IDS`, `MemberIDs`, `ExternalPlatformIDs`, `APIStatuses`, `IPAddresses`, `Countries`, `Roles`
- pagination/time: `Limit`, `Offset`, `SortBy`, `Order`, `GreaterThanID`, `LessThanID`, `CreatedAfter`, `CreatedBefore`, `UpdatedAfter`, `UpdatedBefore`, `StartDate`, `EndDate`
- internal flag: `Export`

Behavior:

- default `Limit` is `100`
- export mode can use config key `user.activity.export.limit`
- if both `ExternalPlatformIDs` and `ProjectFilter` are set, request is rejected
- if `AccessResolver` is configured, non-admin scope is enforced

### `GetEventCategories(ctx)`

Returns distinct non-null event categories.

## Status and Action Constants

Statuses:

- `APIStatusSuccess` (`SUCCESS`)
- `APIStatusDenied` (`DENIED`)
- `APIStatusError` (`ERROR`)

Actions:

- `APIActionRead` (`READ`)
- `APIActionWrite` (`WRITE`)
- `APIActionDelete` (`DELETE`)
- `APIActionUnknown` (`UNKNOWN`)

## Middleware

Both middleware packages follow the same model:

1. create log entry at request start
2. capture status/response
3. update same entry by `session_id`

Packages:

- Gin: `github.com/PayRam/user-activity-go/pkg/useractivity/ginmiddleware`
- net/http: `github.com/PayRam/user-activity-go/pkg/useractivity/httpmiddleware`

Shared config fields:

- `Client *useractivity.Client` (required)
- `Logger *zap.Logger`
- `CaptureRequestBody bool`
- `CaptureResponseBody bool`
- `MaxBodyBytes int64`
- `Redact func([]byte) []byte`
- `ResponseRedact func([]byte) []byte`
- `SkipPaths []string`
- `Skip func(...) bool`
- `SessionIDHeader string`
- `SessionIDFunc func(...) string`
- `IPExtractor func(...) string`
- `GeoLookup *useractivity.GeoLookup` (optional): enriches request with `Country/City/...` from IP
- `CreateEnricher func(..., *useractivity.CreateRequest)`
- `UpdateEnricher func(..., *useractivity.UpdateRequest, *CapturedResponse)`
- `Async bool`
- `OnError func(error)`

Use `SkipPaths` or `Skip` for sensitive routes:

- `/signin`
- `/signup`
- `/oauth/token`
- password reset/change endpoints

### Gin example

```go
router.Use(ginmiddleware.Middleware(ginmiddleware.Config{
	Client:              client,
	CaptureRequestBody:  true,
	CaptureResponseBody: true,
	MaxBodyBytes:        1 * 1024 * 1024,
	GeoLookup:           useractivity.NewGeoLookup(useractivity.GeoLookupConfig{}),
	SkipPaths:           []string{"/signin", "/signup"},
	Redact:              useractivity.RedactDefaultJSONKeys,
	ResponseRedact:      useractivity.RedactDefaultJSONKeys,
	CreateEnricher: func(c *gin.Context, req *useractivity.CreateRequest) {
		// set MemberID / Role / EventName, etc.
	},
	UpdateEnricher: func(c *gin.Context, req *useractivity.UpdateRequest, resp *ginmiddleware.CapturedResponse) {
		// add description / metadata, etc.
	},
}))
```

### net/http example

```go
mw := httpmiddleware.Middleware(httpmiddleware.Config{
	Client:              client,
	CaptureRequestBody:  true,
	CaptureResponseBody: true,
	MaxBodyBytes:        1 * 1024 * 1024,
	GeoLookup:           useractivity.NewGeoLookup(useractivity.GeoLookupConfig{}),
	SkipPaths:           []string{"/signin", "/signup"},
	Redact:              useractivity.RedactDefaultJSONKeys,
	ResponseRedact:      useractivity.RedactDefaultJSONKeys,
	CreateEnricher: func(r *http.Request, req *useractivity.CreateRequest) {
		// set MemberID / Role / EventName, etc.
	},
	UpdateEnricher: func(r *http.Request, req *useractivity.UpdateRequest, resp *httpmiddleware.CapturedResponse) {
		// add description / metadata, etc.
	},
})
```

## Geolocation Helpers

Use built-in helpers when you want geolocation outside middleware hooks.

```go
lookup := useractivity.NewGeoLookup(useractivity.GeoLookupConfig{
	ProviderURLTemplate: "https://ipwhois.app/json/%s", // optional
	ProviderName:        "ipwhois.io",                  // optional
	Timeout:             5 * time.Second,               // optional
	CacheTTL:            24 * time.Hour,                // optional
})

location := lookup.Lookup("8.8.8.8")
if location != nil {
	useractivity.EnrichCreateRequestWithLocation(&createReq, location)
}
```

Environment variables (used if config values are empty):

- `GEOLOCATION_PROVIDER_URL`
- `GEOLOCATION_PROVIDER_NAME`

## Service-Level Tracking

Use service tracker to log business/service/repository operations:

```go
tracker := useractivity.NewServiceTracker(useractivity.ServiceTrackerConfig{
	Client: client,
})

memberID := uint(42)
err := tracker.Track(ctx, useractivity.ServiceOperation{
	Name:      "PaymentService.CreateNewPaymentRequest",
	MemberID:  &memberID,
	APIAction: useractivity.APIActionWrite,
}, func(ctx context.Context) error {
	return repo.Create(ctx, memberID)
})
```

`ServiceOperation` supported fields:

- `Name string`
- `SessionID string`
- `MemberID *uint`
- `ExternalPlatformIDs []uint`
- `Method string`
- `Endpoint string`
- `APIAction string`
- `Description *string`
- `Metadata *string`
- `Role *string`
- `EventCategory *string`
- `EventName *string`

Service metadata written by tracker:

- `source: "SERVICE"`
- `operationName`
- `operationTrail`: ordered list of nested operations

If request middleware already set session id in context, tracker reuses it.

## Metadata Helpers

### `MergeMetadata(existing, patch)`

Use this helper to enrich existing JSON metadata without overwriting all fields.

```go
type apiMeta struct {
	ServiceName   *string `json:"serviceName,omitempty"`
	ServiceStatus *string `json:"serviceStatus,omitempty"`
}

req.Metadata = useractivity.MergeMetadata(req.Metadata, apiMeta{
	ServiceName:   req.EventName,
	ServiceStatus: req.APIStatus,
})
```

If `existing` is non-JSON, it is preserved under `rawMetadata`.

## Redaction and Sensitive Data

Body capture is optional and should be enabled only where needed.

Built-in redaction helper:

- `RedactDefaultJSONKeys` masks keys:
  `password`, `token`, `secret`, `api_key`, `apiKey`, `private_key`, `privateKey`, `access_token`, `refresh_token`, `authorization`, `Authorization`
- matching is case-insensitive and separator-insensitive (`accessToken`, `ACCESS_TOKEN`, `access-token` all match)

Important:

- redaction is JSON key-based
- non-JSON payloads are unchanged
- request/response headers are not captured/redacted by default middleware

## Gin Handlers (Optional)

Package: `github.com/PayRam/user-activity-go/pkg/useractivity/ginhandlers`

Routes:

- `GET /user-activity`
- `GET /user-activity/event-categories`
- `GET /user-activity/export` (CSV)

`HandlerConfig`:

- `Client *useractivity.Client`
- `MemberIDFromContext func(*gin.Context) (uint, bool)`
- `RequireMember bool`
- `ErrorHandler func(*gin.Context, error)`

## Errors

Package-level errors:

- `ErrUnauthorized`
- `ErrBadRequest`

## Migration

Call once during startup:

```go
if err := client.AutoMigrate(ctx); err != nil {
	// handle error
}
```

Table name is `user_activities` unless you set `Config.TablePrefix`.

If your app uses a different table name, set:

- `Config.TableName` to override the base name (example: `activity_logs`)
- `Config.TablePrefix` if needed (example: `core_`)

Example result:

- `TableName: "activity_logs"` => `activity_logs`
- `TablePrefix: "core_", TableName: "activity_logs"` => `core_activity_logs`

## Testing

```bash
go test ./...
go test ./... -cover
```
