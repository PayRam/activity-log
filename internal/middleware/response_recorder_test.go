package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type flushRecorder struct {
	http.ResponseWriter
	flushed bool
}

func (f *flushRecorder) Flush() {
	f.flushed = true
}

type hijackRecorder struct {
	http.ResponseWriter
	hijacked bool
}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

type pushRecorder struct {
	http.ResponseWriter
	pushed bool
}

func (p *pushRecorder) Push(target string, opts *http.PushOptions) error {
	p.pushed = true
	return nil
}

func TestResponseRecorderStatusBody(t *testing.T) {
	base := httptest.NewRecorder()
	recorder := NewResponseRecorder(base, 1024, true)

	if recorder.Status() != http.StatusOK {
		t.Fatalf("expected default status OK, got %d", recorder.Status())
	}

	recorder.WriteHeader(http.StatusCreated)
	if recorder.Status() != http.StatusCreated {
		t.Fatalf("expected status created, got %d", recorder.Status())
	}

	_, _ = recorder.Write([]byte("hello"))
	if recorder.Body() != "hello" {
		t.Fatalf("expected body hello, got %q", recorder.Body())
	}
}

func TestResponseRecorderWriteSetsStatus(t *testing.T) {
	base := httptest.NewRecorder()
	recorder := NewResponseRecorder(base, 1024, true)

	_, _ = recorder.Write([]byte("ok"))
	if recorder.Status() != http.StatusOK {
		t.Fatalf("expected status OK after write, got %d", recorder.Status())
	}
	if recorder.Body() != "ok" {
		t.Fatalf("expected body ok, got %q", recorder.Body())
	}
}

func TestResponseRecorderNoCapture(t *testing.T) {
	base := httptest.NewRecorder()
	recorder := NewResponseRecorder(base, 1024, false)

	_, _ = recorder.Write([]byte("ok"))
	if recorder.Body() != "" {
		t.Fatalf("expected empty body when capture disabled, got %q", recorder.Body())
	}
}

func TestResponseRecorderOptionalInterfaces(t *testing.T) {
	base := httptest.NewRecorder()

	flush := &flushRecorder{ResponseWriter: base}
	recorder := NewResponseRecorder(flush, 1024, false)
	recorder.Flush()
	if !flush.flushed {
		t.Fatalf("expected flush to be called")
	}

	hijack := &hijackRecorder{ResponseWriter: base}
	recorder = NewResponseRecorder(hijack, 1024, false)
	if _, _, err := recorder.Hijack(); err != nil {
		t.Fatalf("unexpected hijack error: %v", err)
	}
	if !hijack.hijacked {
		t.Fatalf("expected hijack to be called")
	}

	push := &pushRecorder{ResponseWriter: base}
	recorder = NewResponseRecorder(push, 1024, false)
	if err := recorder.Push("/test", nil); err != nil {
		t.Fatalf("unexpected push error: %v", err)
	}
	if !push.pushed {
		t.Fatalf("expected push to be called")
	}

	recorder = NewResponseRecorder(base, 1024, false)
	if _, _, err := recorder.Hijack(); err != http.ErrNotSupported {
		t.Fatalf("expected ErrNotSupported for hijack, got %v", err)
	}
	if err := recorder.Push("/test", nil); err != http.ErrNotSupported {
		t.Fatalf("expected ErrNotSupported for push, got %v", err)
	}
	if recorder.Unwrap() != base {
		t.Fatalf("expected unwrap to return base writer")
	}
}
