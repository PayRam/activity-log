# Architecture

## Goal

`activity-log` centralizes activity logging with one reusable library across services.

It captures:

- API request/response lifecycle logs.
- Service-layer operation logs.
- Correlated trails under one `session_id`.

## Layered Design

Public layer:

- `pkg/activitylog`
- `pkg/activitylog/ginmiddleware`
- `pkg/activitylog/httpmiddleware`

Internal layer:

- `internal/services`
- `internal/repositories`
- `internal/models`
- `internal/middleware`
- `internal/utils`

Dependency direction:

- App -> `pkg/activitylog`
- `pkg/activitylog` -> `internal/services`
- `internal/services` -> `internal/repositories`
- `internal/repositories` -> `internal/models`

## Data Flow

### Create flow

1. App/middleware/tracker builds `CreateRequest`.
2. Client validates required fields.
3. Client maps request to repository params.
4. Service validates status code range.
5. Repository inserts `ActivityLog` row.

### Update flow

1. Caller provides `session_id`.
2. Client maps `UpdateRequest` to update params.
3. Repository starts transaction.
4. Repository row-locks by `session_id` (`FOR UPDATE`).
5. Repository applies updates map and reloads row.

### Read flow

1. Client validates request combination rules.
2. Access scope is resolved via `AccessResolver` (optional).
3. Filters are translated to repository query fields.
4. Repository returns rows + total count.
5. Client optionally hydrates members/platforms with resolvers.

## Correlation Strategy

Correlation key is `session_id`.

- Middleware creates/reuses session ID and writes it to context.
- Service tracker reuses context session ID.
- API-level and service-level records can be linked by this value.

## Access Scoping Strategy

`AccessResolver` returns:

- `IsAdmin`
- `AllowedProjectIDs`

Rules for non-admin users:

- Explicit `ProjectIDs`: every requested ID must be allowed.
- No project filter: defaults to allowed project IDs.

## Naming Model

Public API uses `ProjectIDs` and DB column is `project_ids`.

This keeps integration naming closer to product language while preserving storage compatibility.

## Extension Points

- `AccessResolver`
- `ConfigProvider`
- `MemberResolver`
- `ProjectResolver`
- Middleware hooks:
  - `CreateEnricher`
  - `UpdateEnricher`

## Operational Characteristics

- Default read limit: `100`.
- Export limit can be overridden by config key `user.activity.export.limit`.
- Read queries support IDs, session IDs, status code arrays, event arrays, date ranges, sorting.
- Middleware body capture is bounded by max bytes.
- Geolocation lookup is cached in-memory.
