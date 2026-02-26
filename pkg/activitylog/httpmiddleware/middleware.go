package httpmiddleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/PayRam/activity-log/internal/middleware"
	activitylog "github.com/PayRam/activity-log/pkg/activitylog"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContextKey is the type for context keys used by this package.
type ContextKey string

const (
	// SessionIDContextKey stores the generated session ID in the request context.
	SessionIDContextKey ContextKey = "activitylog_session_id"
	// RequestBodyContextKey stores the captured request body string in the request context.
	RequestBodyContextKey ContextKey = "activitylog_request_body"
)

// CapturedResponse contains response details captured by middleware.
type CapturedResponse struct {
	StatusCode int
	Body       string
}

// Config configures the net/http middleware.
type Config struct {
	Client *activitylog.Client
	Logger *zap.Logger

	CaptureRequestBody  bool
	CaptureResponseBody bool
	MaxBodyBytes        int64
	Redact              func([]byte) []byte
	ResponseRedact      func([]byte) []byte

	SkipPaths []string
	Skip      func(*http.Request) bool

	SessionIDHeader string
	SessionIDFunc   func(*http.Request) string
	IPExtractor     func(*http.Request) string

	CreateEnricher func(*http.Request, *activitylog.CreateRequest)
	UpdateEnricher func(*http.Request, *activitylog.UpdateRequest, *CapturedResponse)
	GeoLookup      *activitylog.GeoLookup

	Async   bool
	OnError func(error)
}

// Middleware returns a net/http middleware that logs activity log.
// When called without args, it uses the package default config set by SetDefaultConfig.
func Middleware(configs ...Config) func(http.Handler) http.Handler {
	cfg := resolveConfig(configs)
	if cfg.Client == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	logger := cfg.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	maxBytes := middleware.NormalizeMaxBodyBytes(cfg.MaxBodyBytes)
	skipPaths := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skipPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			sessionID := resolveSessionID(cfg, r)
			ctx := context.WithValue(r.Context(), SessionIDContextKey, sessionID)
			ctx = activitylog.WithSessionID(ctx, sessionID)
			r = r.WithContext(ctx)

			requestBody, err := readRequestBody(cfg, r, maxBytes)
			if err != nil {
				handleError(cfg, logger, err)
			}
			if requestBody != nil {
				ctx = context.WithValue(r.Context(), RequestBodyContextKey, *requestBody)
				r = r.WithContext(ctx)
			}

			ip := resolveIP(cfg, r)
			userAgent := r.UserAgent()
			referer := r.Referer()

			createReq := activitylog.CreateRequest{
				SessionID:   sessionID,
				Method:      r.Method,
				Endpoint:    r.URL.Path,
				APIAction:   middleware.MethodToAction(r.Method),
				APIStatus:   activitylog.APIStatusSuccess,
				RequestBody: requestBody,
			}

			if ip != "" {
				createReq.IPAddress = &ip
				if cfg.GeoLookup != nil {
					activitylog.EnrichCreateRequestWithLocation(&createReq, cfg.GeoLookup.Lookup(ip))
				}
			}
			if userAgent != "" {
				createReq.UserAgent = &userAgent
			}
			if referer != "" {
				createReq.Referer = &referer
			}

			if cfg.CreateEnricher != nil {
				cfg.CreateEnricher(r, &createReq)
			}

			created := true
			if cfg.Async {
				go func() {
					_, err := cfg.Client.CreateActivityLogs(context.Background(), createReq)
					if err != nil {
						handleError(cfg, logger, err)
					}
				}()
			} else {
				if _, err := cfg.Client.CreateActivityLogs(ctx, createReq); err != nil {
					handleError(cfg, logger, err)
					created = false
				}
			}

			recorder := middleware.NewResponseRecorder(w, maxBytes, cfg.CaptureResponseBody)
			next.ServeHTTP(recorder, r)

			if !created && !cfg.Async {
				return
			}

			status := recorder.Status()
			body := recorder.Body()
			apiStatus := activitylog.APIStatus(middleware.StatusToAPIStatus(status))
			method := r.Method
			endpoint := r.URL.Path
			statusCode := activitylog.HTTPStatusCode(status)
			updateReq := activitylog.UpdateRequest{
				SessionID:   sessionID,
				Method:      &method,
				Endpoint:    &endpoint,
				APIStatus:   &apiStatus,
				StatusCode:  &statusCode,
				RequestBody: requestBody,
			}
			if cfg.CaptureResponseBody && body != "" {
				body = redactResponseBody(cfg, body)
				updateReq.ResponseBody = &body
			}

			captured := &CapturedResponse{StatusCode: status, Body: body}
			if cfg.UpdateEnricher != nil {
				cfg.UpdateEnricher(r, &updateReq, captured)
			}

			if cfg.Async {
				go func() {
					_, err := cfg.Client.UpdateActivityLogSessionID(context.Background(), updateReq)
					if err != nil {
						handleError(cfg, logger, err)
					}
				}()
				return
			}

			if _, err := cfg.Client.UpdateActivityLogSessionID(ctx, updateReq); err != nil {
				handleError(cfg, logger, err)
			}
		})
	}
}

func resolveSessionID(cfg Config, r *http.Request) string {
	if cfg.SessionIDFunc != nil {
		if id := cfg.SessionIDFunc(r); id != "" {
			return id
		}
	}
	if cfg.SessionIDHeader != "" {
		if id := strings.TrimSpace(r.Header.Get(cfg.SessionIDHeader)); id != "" {
			return id
		}
	}
	return uuid.NewString()
}

func resolveIP(cfg Config, r *http.Request) string {
	if cfg.IPExtractor != nil {
		return cfg.IPExtractor(r)
	}
	return DefaultIPExtractor(r)
}

func resolveConfig(configs []Config) Config {
	if len(configs) > 0 {
		return configs[0]
	}
	if cfg, ok := loadDefaultConfig(); ok {
		return cfg
	}
	return Config{}
}

// DefaultIPExtractor uses common proxy headers, then RemoteAddr.
func DefaultIPExtractor(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip != "" {
				return ip
			}
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	if r.RemoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func readRequestBody(cfg Config, r *http.Request, maxBytes int64) (*string, error) {
	if !cfg.CaptureRequestBody {
		return nil, nil
	}
	return middleware.ReadRequestBody(r, maxBytes, cfg.Redact)
}

func redactResponseBody(cfg Config, body string) string {
	if body == "" {
		return body
	}
	if cfg.ResponseRedact == nil {
		return body
	}
	redacted := cfg.ResponseRedact([]byte(body))
	if redacted == nil {
		return body
	}
	return string(redacted)
}

func handleError(cfg Config, logger *zap.Logger, err error) {
	if err == nil {
		return
	}
	if cfg.OnError != nil {
		cfg.OnError(err)
		return
	}
	if logger != nil {
		logger.Error("activitylog middleware error", zap.Error(err))
	}
}
