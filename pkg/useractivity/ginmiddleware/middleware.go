package ginmiddleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/PayRam/user-activity-go/internal/middleware"
	"github.com/PayRam/user-activity-go/pkg/useractivity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Context keys stored on gin.Context.
const (
	ContextKeySessionID      = "useractivity_session_id"
	ContextKeyRequestBody    = "useractivity_request_body"
	ContextKeyLibraryEnabled = "useractivity_library_enabled"
)

// CapturedResponse contains response details captured by middleware.
type CapturedResponse struct {
	StatusCode int
	Body       string
}

// Config configures the Gin middleware.
type Config struct {
	Client *useractivity.Client
	Logger *zap.Logger

	CaptureRequestBody  bool
	CaptureResponseBody bool
	MaxBodyBytes        int64
	Redact              func([]byte) []byte
	ResponseRedact      func([]byte) []byte

	SkipPaths []string
	Skip      func(*gin.Context) bool

	SessionIDHeader string
	SessionIDFunc   func(*gin.Context) string
	IPExtractor     func(*gin.Context) string

	CreateEnricher func(*gin.Context, *useractivity.CreateRequest)
	UpdateEnricher func(*gin.Context, *useractivity.UpdateRequest, *CapturedResponse)
	GeoLookup      *useractivity.GeoLookup

	Async   bool
	OnError func(error)
}

// Middleware returns a Gin middleware that logs user activity.
func Middleware(cfg Config) gin.HandlerFunc {
	if cfg.Client == nil {
		return func(c *gin.Context) { c.Next() }
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

	return func(c *gin.Context) {
		if _, ok := skipPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		if cfg.Skip != nil && cfg.Skip(c) {
			c.Next()
			return
		}

		sessionID := resolveSessionID(cfg, c)
		c.Set(ContextKeySessionID, sessionID)
		c.Set(ContextKeyLibraryEnabled, true)
		ctx := context.WithValue(c.Request.Context(), middlewareContextKeySessionID, sessionID)
		ctx = useractivity.WithSessionID(ctx, sessionID)
		c.Request = c.Request.WithContext(ctx)

		requestBody, err := readRequestBody(cfg, c, maxBytes)
		if err != nil {
			handleError(cfg, logger, err)
		}
		if requestBody != nil {
			c.Set(ContextKeyRequestBody, *requestBody)
		}

		ip := resolveIP(cfg, c)
		userAgent := c.Request.UserAgent()
		referer := c.Request.Referer()

		createReq := useractivity.CreateRequest{
			SessionID:   sessionID,
			Method:      c.Request.Method,
			Endpoint:    c.Request.URL.Path,
			APIAction:   middleware.MethodToAction(c.Request.Method),
			APIStatus:   "SUCCESS",
			RequestBody: requestBody,
		}

		if ip != "" {
			createReq.IPAddress = &ip
			if cfg.GeoLookup != nil {
				useractivity.EnrichCreateRequestWithLocation(&createReq, cfg.GeoLookup.Lookup(ip))
			}
		}
		if userAgent != "" {
			createReq.UserAgent = &userAgent
		}
		if referer != "" {
			createReq.Referer = &referer
		}

		if cfg.CreateEnricher != nil {
			cfg.CreateEnricher(c, &createReq)
		}

		created := true
		if cfg.Async {
			go func() {
				_, err := cfg.Client.Create(context.Background(), createReq)
				if err != nil {
					handleError(cfg, logger, err)
				}
			}()
		} else {
			if _, err := cfg.Client.Create(ctx, createReq); err != nil {
				handleError(cfg, logger, err)
				created = false
			}
		}

		recorder := newGinBodyWriter(c.Writer, maxBytes, cfg.CaptureResponseBody)
		c.Writer = recorder

		c.Next()

		if !created && !cfg.Async {
			return
		}

		status := recorder.Status()
		body := recorder.Body()
		apiStatus := middleware.StatusToAPIStatus(status)
		updateReq := useractivity.UpdateRequest{
			SessionID:  sessionID,
			APIStatus:  &apiStatus,
			StatusCode: &status,
		}
		if cfg.CaptureResponseBody && body != "" {
			body = redactResponseBody(cfg, body)
			updateReq.ResponseBody = &body
		}

		captured := &CapturedResponse{StatusCode: status, Body: body}
		if cfg.UpdateEnricher != nil {
			cfg.UpdateEnricher(c, &updateReq, captured)
		}

		if cfg.Async {
			go func() {
				_, err := cfg.Client.Update(context.Background(), updateReq)
				if err != nil {
					handleError(cfg, logger, err)
				}
			}()
			return
		}

		if _, err := cfg.Client.Update(ctx, updateReq); err != nil {
			handleError(cfg, logger, err)
		}
	}
}

type contextKey string

const middlewareContextKeySessionID contextKey = "useractivity_session_id"

func resolveSessionID(cfg Config, c *gin.Context) string {
	if cfg.SessionIDFunc != nil {
		if id := cfg.SessionIDFunc(c); id != "" {
			return id
		}
	}
	if cfg.SessionIDHeader != "" {
		if id := strings.TrimSpace(c.GetHeader(cfg.SessionIDHeader)); id != "" {
			return id
		}
	}
	return uuid.NewString()
}

func resolveIP(cfg Config, c *gin.Context) string {
	if cfg.IPExtractor != nil {
		return cfg.IPExtractor(c)
	}
	return c.ClientIP()
}

func readRequestBody(cfg Config, c *gin.Context, maxBytes int64) (*string, error) {
	if !cfg.CaptureRequestBody {
		return nil, nil
	}
	return middleware.ReadRequestBody(c.Request, maxBytes, cfg.Redact)
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
		logger.Error("useractivity middleware error", zap.Error(err))
	}
}

// gin body writer

type ginBodyWriter struct {
	gin.ResponseWriter
	status  int
	capture *middleware.BodyCapture
}

func newGinBodyWriter(w gin.ResponseWriter, maxBodyBytes int64, captureBody bool) *ginBodyWriter {
	var capture *middleware.BodyCapture
	if captureBody {
		capture = middleware.NewBodyCapture(maxBodyBytes)
	}
	return &ginBodyWriter{ResponseWriter: w, capture: capture}
}

func (w *ginBodyWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ginBodyWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.capture != nil {
		w.capture.Write(p)
	}
	return w.ResponseWriter.Write(p)
}

func (w *ginBodyWriter) WriteString(s string) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.capture != nil {
		w.capture.Write([]byte(s))
	}
	return w.ResponseWriter.WriteString(s)
}

func (w *ginBodyWriter) Status() int {
	if w.status == 0 {
		if w.ResponseWriter != nil {
			if status := w.ResponseWriter.Status(); status != 0 {
				return status
			}
		}
		return http.StatusOK
	}
	return w.status
}

func (w *ginBodyWriter) Body() string {
	if w.capture == nil {
		return ""
	}
	return w.capture.String()
}
