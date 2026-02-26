package middleware

import (
	"bufio"
	"net"
	"net/http"
)

// ResponseRecorder wraps http.ResponseWriter to capture status and body.
type ResponseRecorder struct {
	http.ResponseWriter
	status  int
	capture *BodyCapture
}

func NewResponseRecorder(w http.ResponseWriter, maxBodyBytes int64, captureBody bool) *ResponseRecorder {
	var capture *BodyCapture
	if captureBody {
		capture = NewBodyCapture(maxBodyBytes)
	}
	return &ResponseRecorder{
		ResponseWriter: w,
		capture:        capture,
	}
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if r.capture != nil {
		r.capture.Write(p)
	}
	return r.ResponseWriter.Write(p)
}

func (r *ResponseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *ResponseRecorder) Body() string {
	if r.capture == nil {
		return ""
	}
	return r.capture.String()
}

// Ensure optional interfaces are preserved.
func (r *ResponseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (r *ResponseRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (r *ResponseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
