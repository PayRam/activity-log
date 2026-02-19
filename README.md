# activity-log

Go library for persisting activity log logs with:

- direct APIs: `Create`, `Update`, `Get`, `GetEventCategories`
- optional HTTP middleware: Gin and `net/http`
- optional service-level tracker for business/repository operations
- metadata merge, JSON redaction, and geolocation helpers
- no built-in HTTP handlers or runnable app entrypoint

Repository: `https://github.com/PayRam/activity-log`  
Module path: `github.com/PayRam/activity-log`

## Install

```bash
go get github.com/PayRam/activity-log
```

## Quick Start

```go
package main

import (
	"context"
	"net/http"

	"github.com/PayRam/activity-log/pkg/activitylog"
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

	client, err := activitylog.New(activitylog.Config{
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

	_, err = client.CreateActivityLogs(context.Background(), activitylog.CreateRequest{
		SessionID: "session-123",
		Method:    "POST",
		Endpoint:  "/api/v1/payment-request",
		APIAction: activitylog.APIActionWrite,
		APIStatus: activitylog.APIStatusSuccess,
	})
	if err != nil {
		return err
	}

	status := activitylog.APIStatusError
	code := activitylog.HTTPStatusCode(http.StatusInternalServerError)
	msg := "downstream failed"
	_, err = client.UpdateActivityLogSessionID(context.Background(), activitylog.UpdateRequest{
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
activity-log/
├── pkg/
│   └── activitylog/                # public API surface
│       ├── activity_log.go         # Client + Create/Update/Get APIs
│       ├── types.go                 # request/response contracts
│       ├── service_tracker.go       # service/repository operation tracker
│       ├── geolocation.go           # public geolocation API + enrichers
│       ├── metadata.go              # public metadata merge helper
│       ├── redact.go                # public redaction helper
│       ├── ginmiddleware/           # optional Gin adapter middleware
│       └── httpmiddleware/          # optional net/http adapter middleware
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

- app imports `pkg/activitylog` (and optional `pkg/activitylog/ginmiddleware`, `pkg/activitylog/httpmiddleware`)
- `pkg/activitylog` uses `internal/services`
- `internal/services` uses `internal/repositories`
- `internal/repositories` uses `internal/models` + `gorm`
- `internal/*` packages are implementation details and not for external imports

## Client Config

`activitylog.Config`:

- `DB *gorm.DB` (required): gorm database handle
- `Logger *zap.Logger` (optional): defaults to production zap logger
- `TablePrefix string` (optional): prefixes `activity_logs` table name
- `TableName string` (optional): overrides base table name (for example `activity_logs`)
- `EventDeriver EventDeriver` (optional): derives `EventCategory`/`EventName` on `Create` when those fields are missing
- `EventInfoDeriver EventInfoDeriver` (optional): derives `EventCategory`/`EventName`/`Description` on `Create` and `Update` when those fields are missing
- `AccessResolver AccessResolver` (optional): applies `Get` access scoping
- `ConfigProvider ConfigProvider` (optional): can override export limit (`user.activity.export.limit`)
- `MemberResolver MemberResolver` (optional): hydrates `Activity.Member` in `Get`
- `ProjectResolver ProjectResolver` (optional): hydrates `Activity.Projects` in `Get`

## Environment Variables

- `ACTIVITY_LOG_TEST_POSTGRES_DSN`: PostgreSQL DSN used by integration tests that need a real database
- `GEOLOCATION_PROVIDER_URL`: geolocation provider URL template fallback (used when `GeoLookupConfig.ProviderURLTemplate` is empty)
- `GEOLOCATION_PROVIDER_NAME`: geolocation provider name fallback (used when `GeoLookupConfig.ProviderName` is empty)

## Integrating with Project Naming

Common mapping:

| Library field | Your app (example) |
| --- | --- |
| `ProjectIDs` | `ProjectIDs` |
| `ProjectResolver` | `ProjectResolver` |
| `AccessContext.AllowedProjectIDs` | IDs of projects user can access |
| `ProjectFilter` | project-scope filter behavior |

Practical integration rule:

- Treat `ProjectIDs` as the scope IDs from your domain model.
- Keep your domain naming internally.
- Add a thin adapter only at integration boundaries.

Example adapter setup:

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

func (r *projectResolver) GetByIDs(ctx context.Context, ids []uint) (map[uint]activitylog.ProjectInfo, error) {
	projects, err := r.projectService.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uint]activitylog.ProjectInfo, len(projects))
	for id, p := range projects {
		out[id] = activitylog.ProjectInfo{
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

func (r *projectAccessResolver) Resolve(ctx context.Context, memberID uint) (*activitylog.AccessContext, error) {
	allowed, err := r.projectService.GetAllowedProjectIDs(ctx, memberID)
	if err != nil {
		return nil, err
	}
	return &activitylog.AccessContext{
		IsAdmin:           false,
		AllowedProjectIDs: allowed,
	}, nil
}

client, err := activitylog.New(activitylog.Config{
	DB:              db,
	ProjectResolver: &projectResolver{projectService: projectService},
	AccessResolver:  &projectAccessResolver{projectService: projectService},
})
```

Request mapping example:

```go
projectID := uint(101)
_, err := client.CreateActivityLogs(ctx, activitylog.CreateRequest{
	SessionID:           "s-1",
	Method:              "POST",
	Endpoint:            "/payments",
	APIAction:           activitylog.APIActionWrite,
	APIStatus:           activitylog.APIStatusSuccess,
	ProjectIDs: []uint{projectID}, // project scope in your app
})
```

## API Reference

### `CreateActivityLogs(ctx, CreateRequest)` (store)

Required fields:

- `SessionID string`
- `Method string`
- `Endpoint string`
- `APIAction string`
- `APIStatus APIStatus`

Supported optional fields:

- actor scope: `MemberID *uint`, `ProjectIDs []uint`
- result: `StatusCode *HTTPStatusCode`, `Description *string`, `APIErrorMsg *string`
- request info: `IPAddress *string`, `UserAgent *string`, `Referer *string`
- payloads: `RequestBody *string`, `ResponseBody *string`, `Metadata *string`
- classification: `Role *string`, `EventCategory *string`, `EventName *string`
- geolocation: `Country *string`, `CountryCode *string`, `Region *string`, `City *string`, `Timezone *string`, `Latitude *float64`, `Longitude *float64`

Notes:

- `Endpoint` is stored in DB field `api_part`.
- `ProjectIDs` is stored as JSON/JSONB array.
- if `EventCategory` / `EventName` are not provided, library falls back to URL segment after `/api/v1/` (for example `/api/v1/payment-request` -> `payment-request`)
- if `Config.EventInfoDeriver` is provided, it is used first for category/name/description fallback
- if `Config.EventDeriver` is provided, it is used for category/name fallback when event info deriver does not provide those values

Event deriver options:

- `DefaultEventDeriver`: default endpoint-based fallback
- `DefaultEventInfoDeriver`: default endpoint/method/status-based fallback including description
- `NewCoreLikeEventDeriver`: helper that approximates `test/core` `deriveEventInfo` style (`CATEGORY_ACTION`)
- `NewCoreLikeEventInfoDeriver`: helper that approximates `test/core` `deriveEventInfo` style (`CATEGORY_ACTION`) and description text

Example:

```go
client, err := activitylog.New(activitylog.Config{
	DB: db,
	EventInfoDeriver: activitylog.NewCoreLikeEventInfoDeriver(activitylog.CoreLikeEventDeriverConfig{
		BasePath:   "/api/v1",
		TableNames: []string{"members", "payment_requests", "withdrawals"},
	}),
})
```

### `UpdateActivityLogSessionID(ctx, UpdateRequest)`

Required field:

- `SessionID string`

Supported updatable fields:

- actor scope: `MemberID *uint`, `ProjectIDs *[]uint`
- route/action/status: `Method *string`, `Endpoint *string`, `APIAction *string`, `APIStatus *APIStatus`
- result: `StatusCode *HTTPStatusCode`, `Description *string`, `APIErrorMsg *string`
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
| `StatusCode` | `*HTTPStatusCode` | `O` | `O` | Use `net/http` constants (for example `http.StatusOK`) |
| `Description` | `*string` | `O` | `O` | Human-readable message |
| `APIErrorMsg` | `*string` | `O` | `O` | Error text if any |

Actor and classification:

| Field | Type | Create | Update | Notes |
| --- | --- | --- | --- | --- |
| `MemberID` | `*uint` | `O` | `O` | Actor member id |
| `ProjectIDs` | `[]uint` / `*[]uint` | `O` | `O` | Create uses `[]uint`; Update uses `*[]uint` for nullable semantics |
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
- update `ProjectIDs == nil`: no change
- update `ProjectIDs != nil` and `*ProjectIDs == nil`: set DB value to `NULL`
- update `ProjectIDs != nil` and `len(*ProjectIDs) == 0`: set empty JSON array (`[]`)
- update `ProjectIDs != nil` and populated: set provided IDs

Implementation detail:

- update is by `session_id`, inside a DB transaction with row lock
- GORM struct updates omit zero values for non-pointer fields

### `GetActivityLogs(ctx, memberID, GetRequest)`

Supported filters:

- arrays: `StatusCodes` (query key: `statusCode` repeated), `EventCategories`, `Methods`, `EventNames`, `IDS`, `MemberIDs`, `ProjectIDs`, `SessionIDs`, `APIStatuses`, `IPAddresses`, `Countries`, `Roles`
- exact/single: `Search`, `ProjectFilter`
- pagination/time: `Limit`, `Offset`, `SortBy`, `Order`, `GreaterThanID`, `LessThanID`, `CreatedAfter`, `CreatedBefore`, `UpdatedAfter`, `UpdatedBefore`, `StartDate`, `EndDate`
- internal flag: `Export`

Behavior:

- default `Limit` is `100`
- export mode can use config key `user.activity.export.limit`
- if both `ProjectIDs` and `ProjectFilter` are set, request is rejected
- if `AccessResolver` is configured, non-admin scope is enforced
- supported `ProjectFilter` values are `ALL` and `NO_IDS`
- unknown `ProjectFilter` values are rejected as unauthorized

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

- Gin: `github.com/PayRam/activity-log/pkg/activitylog/ginmiddleware`
- net/http: `github.com/PayRam/activity-log/pkg/activitylog/httpmiddleware`

Shared config fields:

- `Client *activitylog.Client` (required)
- `Logger *zap.Logger`
- `CaptureRequestBody bool`
- `CaptureResponseBody bool`
- `MaxBodyBytes int64`
- `Redact func([]byte) []byte`
- `ResponseRedact func([]byte) []byte`
- `SkipPaths []string`
- `Skip func(*gin.Context) bool` (Gin)
- `Skip func(*http.Request) bool` (net/http)
- `SessionIDHeader string`
- `SessionIDFunc func(*gin.Context) string` (Gin)
- `SessionIDFunc func(*http.Request) string` (net/http)
- `IPExtractor func(*gin.Context) string` (Gin)
- `IPExtractor func(*http.Request) string` (net/http)
- `GeoLookup *activitylog.GeoLookup` (optional): enriches request with `Country/City/...` from IP
- `CreateEnricher func(*gin.Context, *activitylog.CreateRequest)` (Gin)
- `CreateEnricher func(*http.Request, *activitylog.CreateRequest)` (net/http)
- `UpdateEnricher func(*gin.Context, *activitylog.UpdateRequest, *ginmiddleware.CapturedResponse)` (Gin)
- `UpdateEnricher func(*http.Request, *activitylog.UpdateRequest, *httpmiddleware.CapturedResponse)` (net/http)
- `Async bool`
- `OnError func(error)`

Optional global wrapper (same shape as `Middleware()` with no args):

- Gin: call `ginmiddleware.SetDefaultConfig(cfg)` once, then use `ginmiddleware.Middleware()`
- net/http: call `httpmiddleware.SetDefaultConfig(cfg)` once, then use `httpmiddleware.Middleware()`
- call `ResetDefaultConfig()` in tests to avoid cross-test leakage

Important: this global wrapper trades convenience for weaker test isolation. Prefer explicit DI (`Middleware(Config{...})`) for production wiring.

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
	GeoLookup:           activitylog.NewGeoLookup(activitylog.GeoLookupConfig{}),
	SkipPaths:           []string{"/signin", "/signup"},
	Redact:              activitylog.RedactDefaultJSONKeys,
	ResponseRedact:      activitylog.RedactDefaultJSONKeys,
	CreateEnricher: func(c *gin.Context, req *activitylog.CreateRequest) {
		// set MemberID / Role / EventName, etc.
	},
	UpdateEnricher: func(c *gin.Context, req *activitylog.UpdateRequest, resp *ginmiddleware.CapturedResponse) {
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
	GeoLookup:           activitylog.NewGeoLookup(activitylog.GeoLookupConfig{}),
	SkipPaths:           []string{"/signin", "/signup"},
	Redact:              activitylog.RedactDefaultJSONKeys,
	ResponseRedact:      activitylog.RedactDefaultJSONKeys,
	CreateEnricher: func(r *http.Request, req *activitylog.CreateRequest) {
		// set MemberID / Role / EventName, etc.
	},
	UpdateEnricher: func(r *http.Request, req *activitylog.UpdateRequest, resp *httpmiddleware.CapturedResponse) {
		// add description / metadata, etc.
	},
})
```

## Geolocation Helpers

Use built-in helpers when you want geolocation outside middleware hooks.

`GeoLookupConfig` fields:

- `ProviderURLTemplate string`
- `ProviderName string`
- `Timeout time.Duration`
- `CacheTTL time.Duration`
- `Logger *zap.Logger`
- `HTTPClient *http.Client`

```go
lookup := activitylog.NewGeoLookup(activitylog.GeoLookupConfig{
	ProviderURLTemplate: "https://ipwhois.app/json/%s", // optional
	ProviderName:        "ipwhois.io",                  // optional
	Timeout:             5 * time.Second,               // optional
	CacheTTL:            24 * time.Hour,                // optional
})

location := lookup.Lookup("8.8.8.8")
if location != nil {
	activitylog.EnrichCreateRequestWithLocation(&createReq, location)
}
```

Environment variables (used if config values are empty):

- `GEOLOCATION_PROVIDER_URL`
- `GEOLOCATION_PROVIDER_NAME`

## Service-Level Tracking

Use service tracker to log business/service/repository operations:

```go
tracker := activitylog.NewServiceTracker(activitylog.ServiceTrackerConfig{
	Client: client,
})

memberID := uint(42)
err := tracker.Track(ctx, activitylog.ServiceOperation{
	Name:      "PaymentService.CreateNewPaymentRequest",
	MemberID:  &memberID,
	APIAction: activitylog.APIActionWrite,
}, func(ctx context.Context) error {
	return repo.CreateActivityLogs(ctx, memberID)
})
```

`ServiceTrackerConfig` fields:

- `Client *activitylog.Client` (required)
- `Logger *zap.Logger`
- `Async bool`
- `OnError func(error)`
- `CreateEnricher func(context.Context, *activitylog.CreateRequest)`
- `UpdateEnricher func(context.Context, *activitylog.UpdateRequest, *activitylog.ServiceResult)`

`ServiceOperation` supported fields:

- `Name string`
- `SessionID string`
- `MemberID *uint`
- `ProjectIDs []uint`
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

req.Metadata = activitylog.MergeMetadata(req.Metadata, apiMeta{
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

Table name is `activity_logs` unless you set `Config.TablePrefix`.

If your app uses a different table name, set:

- `Config.TableName` to override the base name (example: `activity_logs`)
- `Config.TablePrefix` if needed (example: `core_`)

Example result:

- `TableName: "activity_logs"` => `activity_logs`
- `TablePrefix: "core_", TableName: "activity_logs"` => `core_activity_logs`

## Testing

Integration tests that touch PostgreSQL require:

- `ACTIVITY_LOG_TEST_POSTGRES_DSN`

```bash
go test ./...
go test ./... -cover
```
